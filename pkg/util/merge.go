package util

// MergeStrings merge string slices
func MergeStrings(a, b []string) []string {
	for _, bv := range b {
		found := false
		for _, av := range a {
			if av == bv {
				found = true
				break
			}
		}
		if !found {
			a = append(a, bv)
		}
	}
	return a
}

// InterfaceSlice helper function to convert []string to []interface{}
// see https://github.com/golang/go/wiki/InterfaceSlice
func InterfaceSlice(slice []string) []interface{} {
	islice := make([]interface{}, len(slice))
	for i, v := range slice {
		islice[i] = v
	}
	return islice
}
