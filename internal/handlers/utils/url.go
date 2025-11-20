package utils

// splitURLPath splits a URL path by "/" and returns the parts
func splitURLPath(path string) []string {
	if path == "" {
		return []string{}
	}

	// Remove leading slash if present
	if path[0] == '/' {
		path = path[1:]
	}

	// Split by "/"
	parts := make([]string, 0)
	current := ""
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	// Add the last part if it exists
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
