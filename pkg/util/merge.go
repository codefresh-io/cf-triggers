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
