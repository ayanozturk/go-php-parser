package lexer

// readLineComment slices the line comment directly from input (zero allocation).
// commentStart is the byte offset of the first '/'.
// Called with l.char at the second '/'.
func (l *Lexer) readLineComment(commentStart int) string {
	l.readChar() // move past second '/'
	for l.char != '\n' && l.char != 0 {
		l.readChar()
	}
	return l.input[commentStart:l.pos]
}

// readHashComment slices the hash comment directly from input (zero allocation).
// Called with l.char at '#'.
func (l *Lexer) readHashComment() string {
	commentStart := l.pos
	l.readChar() // move past '#'
	for l.char != '\n' && l.char != 0 {
		l.readChar()
	}
	return l.input[commentStart:l.pos]
}

// readBlockComment slices the block comment directly from input (zero allocation).
// commentStart is the byte offset of the opening '/'.
// Called with l.char at the first (or second, for /**) '*'.
func (l *Lexer) readBlockComment(commentStart int) string {
	l.readChar() // move past '*'
	for {
		if l.char == 0 {
			break
		}
		if l.char == '*' && l.peekChar() == '/' {
			l.readChar() // consume '*'
			l.readChar() // consume '/'
			break
		}
		l.readChar()
	}
	return l.input[commentStart:l.pos]
}
