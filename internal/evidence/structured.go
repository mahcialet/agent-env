package evidence

import (
	"reflect"
	"strings"
)

// ContainsSecretString checks string data in an acyclic configuration value.
// Struct field names and serialization tags are schema, not user data; dynamic
// map keys are user data and are checked along with their values.
func ContainsSecretString(value any, secrets []string) bool {
	var visit func(reflect.Value) bool
	visit = func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.String:
			text := v.String()
			for _, secret := range secrets {
				if secret != "" && strings.Contains(text, secret) {
					return true
				}
			}
		case reflect.Interface, reflect.Pointer:
			return !v.IsNil() && visit(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if visit(v.Field(i)) {
					return true
				}
			}
		case reflect.Map:
			entries := v.MapRange()
			for entries.Next() {
				if visit(entries.Key()) || visit(entries.Value()) {
					return true
				}
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				if visit(v.Index(i)) {
					return true
				}
			}
		}
		return false
	}
	return visit(reflect.ValueOf(value))
}
