package helper

// MethodNameSpanAfterFunction finds the method name after a `function` keyword.
// functionCol is 1-based column of `f` in `function` on `line`.
// Returns 1-based half-open [startCol, endCol) for `name`, ok=false if not found.
func MethodNameSpanAfterFunction(line string, functionCol int, name string) (startCol, endCol int, ok bool) {
	if functionCol < 1 || name == "" {
		return 0, 0, false
	}

	idx := functionCol - 1
	if idx >= len(line) {
		return 0, 0, false
	}

	// Scan for "function" starting at idx if not already there.
	if idx+len("function") <= len(line) && line[idx:idx+len("function")] == "function" {
		idx += len("function")
	} else {
		found := false
		for i := idx; i+len("function") <= len(line); i++ {
			if line[i:i+len("function")] == "function" {
				idx = i + len("function")
				found = true
				break
			}
		}
		if !found {
			return 0, 0, false
		}
	}

	for idx < len(line) && (line[idx] == ' ' || line[idx] == '\t') {
		idx++
	}

	if idx < len(line) && line[idx] == '&' {
		idx++
		for idx < len(line) && (line[idx] == ' ' || line[idx] == '\t') {
			idx++
		}
	}

	if idx+len(name) > len(line) || line[idx:idx+len(name)] != name {
		return 0, 0, false
	}

	startCol = idx + 1
	endCol = startCol + len(name)
	return startCol, endCol, true
}
