package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var ErrTransientInput = errors.New("remote text input requires a non-persistent transport and is not supported; use local mode")

// ValidateDurablePayload protects transient input fields before an operation is
// written to either durable queue. Scan tokens rather than a map so duplicate
// JSON members cannot hide a value retained in the original payload bytes.
func ValidateDurablePayload(kind string, payload json.RawMessage) error {
	if kind != "ui" && kind != "browser" {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(payload))
	d.UseNumber()
	var value func(string) error
	value = func(key string) error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		if strings.EqualFold(key, "text") && token != nil && token != "" {
			return ErrTransientInput
		}
		if strings.EqualFold(key, "operation") {
			if op, ok := token.(string); ok && strings.EqualFold(op, "set-text") {
				return ErrTransientInput
			}
		}
		switch token {
		case json.Delim('{'):
			for d.More() {
				field, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := field.(string)
				if !ok {
					return errors.New("invalid operation object")
				}
				if err := value(name); err != nil {
					return err
				}
			}
			_, err = d.Token()
		case json.Delim('['):
			for d.More() {
				if err := value(""); err != nil {
					return err
				}
			}
			_, err = d.Token()
		}
		return err
	}
	if err := value(""); err != nil {
		// Syntax errors can quote input; never return sensitive payload excerpts.
		if errors.Is(err, ErrTransientInput) {
			return err
		}
		return errors.New("invalid durable operation payload")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("invalid durable operation payload")
	}
	return nil
}
