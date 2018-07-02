package util

import (
	"fmt"
	"strings"
)

// InterfaceSlice helper function to convert []string to []interface{}
// see https://github.com/golang/go/wiki/InterfaceSlice
func InterfaceSlice(slice []string) []interface{} {
	islice := make([]interface{}, len(slice))
	for i, v := range slice {
		islice[i] = v
	}
	return islice
}

// StringSliceToMap convert string slice (with key=value strings) to map
func StringSliceToMap(values []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, v := range values {
		kv := strings.Split(v, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("unexpected 'value: %s ; should be in 'key=value' form", v)
		}
		result[kv[0]] = kv[1]
	}
	return result, nil
}
