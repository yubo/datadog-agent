package util

//Dedupe deduplicates elements in a string slice
func Dedupe(s []string) []string {
	m := make(map[string]struct{})

	for _, e := range s {
		m[e] = struct{}{}
	}

	i := 0
	d := make([]string, len(m))
	for e := range m {
		d[i] = e
		i++
	}

	return d
}
