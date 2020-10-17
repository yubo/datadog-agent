package util

//Dedupe deduplicates elements in a string slice
func Dedupe(s []string) []string {
	m := make(map[string]struct{})
	d := []string{}

	for _, e := range s {
		m[e] = struct{}{}
	}

	for e := range m {
		d = append(d, e)
	}

	return d
}
