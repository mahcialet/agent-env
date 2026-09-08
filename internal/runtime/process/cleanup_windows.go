package process

import (
	"errors"

	"golang.org/x/sys/windows"
)

func retryableStateRemoval(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
