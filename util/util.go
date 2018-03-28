package util

// creates a map with entries whose keys are array elements and value empty struct
// key=array[index] value=<empty_struct>
func Array2map(arr ...string) map[string]struct{} {

	var result map[string]struct{}

	if length := len(arr); length > 0 {
		result = make(map[string]struct{}, length)

		for _, path := range arr {
			result[path] = struct{}{}
		}
	}

	return result
}
