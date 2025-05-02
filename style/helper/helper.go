package helper

// QuoteState tracks whether we are inside single or double quotes
type QuoteState struct {
	InSingle bool
	InDouble bool
}

// CommentState tracks whether we are inside a block comment or heredoc
type CommentState struct {
	InBlockComment bool
	InHeredoc      bool
	HeredocEnd     string
}

// SkipLineComment returns true if the line is a pure line comment
func SkipLineComment(line string) bool {
	if len(line) == 0 {
		return true
	}
	if len(line) > 1 && (line[0:2] == "//" || line[0:1] == "#") {
		return true
	}
	return false
}

// HandleBlockComment updates state if entering or exiting a block comment
func HandleBlockComment(line string, j int, state *CommentState) int {
	if state.InBlockComment {
		if j+1 < len(line) && line[j] == '*' && line[j+1] == '/' {
			state.InBlockComment = false
			return j + 2
		}
		return j + 1
	}
	if j+1 < len(line) && line[j] == '/' && line[j+1] == '*' {
		state.InBlockComment = true
		return j + 2
	}
	return j
}

// HandleHeredocStart checks for heredoc/nowdoc start and updates state
func HandleHeredocStart(line string, j int, state *CommentState) int {
	if j+2 < len(line) && line[j] == '<' && line[j+1] == '<' && line[j+2] == '<' {
		k := j + 3
		for k < len(line) && (line[k] == ' ' || line[k] == '\t') {
			k++
		}
		start := k
		for k < len(line) && ((line[k] >= 'a' && line[k] <= 'z') || (line[k] >= 'A' && line[k] <= 'Z') || (line[k] >= '0' && line[k] <= '9') || line[k] == '_' || line[k] == '\'' || line[k] == '"') {
			k++
		}
		state.HeredocEnd = line[start:k]
		if state.HeredocEnd != "" {
			state.InHeredoc = true
			return k
		}
	}
	return j
}

// HandleHeredocEnd checks for heredoc/nowdoc end and updates state
func HandleHeredocEnd(line string, state *CommentState) bool {
	if state.InHeredoc {
		k := 0
		for k < len(line) && (line[k] == ' ' || line[k] == '\t') {
			k++
		}
		if state.HeredocEnd != "" && line[k:] == state.HeredocEnd {
			state.InHeredoc = false
			state.HeredocEnd = ""
		}
		return true
	}
	return false
}

// HandleQuotes updates quote state
func HandleQuotes(line string, j int, qs *QuoteState) int {
	if !qs.InDouble && line[j] == '\'' {
		qs.InSingle = !qs.InSingle
		return j + 1
	}
	if !qs.InSingle && line[j] == '"' {
		qs.InDouble = !qs.InDouble
		return j + 1
	}
	if line[j] == '\\' {
		return j + 2
	}
	return j
}
