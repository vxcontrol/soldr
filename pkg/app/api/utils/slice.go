package utils

// StringInSlice is function to lookup string value in slice of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// StringsInSlice is function to lookup all strings in slice of strings
func StringsInSlice(a []string, list []string) bool {
	for _, b := range a {
		if !StringInSlice(b, list) {
			return false
		}
	}
	return true
}
