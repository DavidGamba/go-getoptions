package getoptions

// stringSliceIndex - indicates if an element is found in the slice and what its index is
func stringSliceIndex(ss []string, e string) (int, bool) {
	for i, s := range ss {
		if s == e {
			return i, true
		}
	}
	return -1, false
}
