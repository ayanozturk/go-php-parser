package lexer

import (
	"github.com/ayanozturk/go-php-parser/token"
	"strings"
	"unicode/utf8"
)

var asciiStrings [128]string

func init() {
	for i := 0; i < 128; i++ {
		asciiStrings[i] = string(rune(i))
	}
}

func asciiString(c rune) string {
	if c >= 0 && c < 128 {
		return asciiStrings[c]
	}
	return string(c)
}

type Lexer struct {
	input    string
	pos      int
	readPos  int
	char     rune // Unicode-aware current character
	size     int  // Size of last rune read
	line     int
	column   int
	inString bool // Tracks if currently inside a string
	// For heredoc token queue
	heredocTokens []token.Token
	// Lookahead cache: avoids state save/restore on PeekToken
	hasPeeked   bool
	peekedToken token.Token
	// inHTML tracks whether the lexer is currently scanning literal
	// (non-PHP) content outside of <?php ... ?> tags. PHP files may start
	// with, contain, or end with inline HTML/text; this flag switches the
	// scanner between raw-text mode and normal PHP tokenization.
	inHTML bool
}

// inStringMode returns whether the lexer is currently inside a string.
func (l *Lexer) inStringMode() bool {
	return l.inString
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next rune from input and advances position, supporting Unicode.
func (l *Lexer) readChar() {
	// line and column describe the rune being loaded. Advance from the
	// previous rune before decoding the next one so the first rune of both the
	// file and every subsequent line is reported at column 1.
	if l.readPos == 0 {
		l.column = 1
	} else if l.char == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	if l.readPos >= len(l.input) {
		l.char = 0
		l.size = 0
	} else {
		c := l.input[l.readPos]
		if c < utf8.RuneSelf {
			l.char = rune(c)
			l.size = 1
		} else {
			l.char, l.size = utf8.DecodeRuneInString(l.input[l.readPos:])
		}
	}
	l.pos = l.readPos
	l.readPos += l.size
}

// peekChar peeks the next rune without advancing position (Unicode-aware).
func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	c := l.input[l.readPos]
	if c < utf8.RuneSelf {
		return rune(c)
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return r
}

// SkipBalancedCurlyBlock advances from the current "{" through its matching
// "}". It is used by symbol-only parser passes that need declarations and
// signatures but not statement bodies.
func (l *Lexer) SkipBalancedCurlyBlock() bool {
	depth := 0
	if l.char != '{' {
		depth = 1
	}
	for l.char != 0 {
		switch l.char {
		case '\'', '"':
			l.skipQuotedString(l.char)
			continue
		case '/':
			switch l.peekChar() {
			case '/':
				l.skipLineComment()
				continue
			case '*':
				l.skipBlockComment()
				continue
			}
		case '#':
			l.skipLineComment()
			continue
		case '{':
			depth++
		case '}':
			depth--
			l.readChar()
			if depth == 0 {
				return true
			}
			continue
		}
		l.readChar()
	}
	return false
}

func (l *Lexer) skipQuotedString(quote rune) {
	l.readChar()
	for l.char != 0 {
		if l.char == '\\' {
			l.readChar()
			if l.char != 0 {
				l.readChar()
			}
			continue
		}
		if l.char == quote {
			l.readChar()
			return
		}
		l.readChar()
	}
}

func (l *Lexer) skipLineComment() {
	for l.char != 0 && l.char != '\n' {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar()
	l.readChar()
	for l.char != 0 {
		if l.char == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return
		}
		l.readChar()
	}
}

func (l *Lexer) skipWhitespace() {
	for l.char == ' ' || l.char == '\t' || l.char == '\n' || l.char == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readString(quote byte) string {
	// Fast path: scan forward to see if we can slice directly without escapes or newlines
	hasEscapesOrNewlines := false
	end := l.pos
	for end < len(l.input) {
		c := l.input[end]
		if c == quote {
			break
		}
		if c == '\\' || c == '\n' || c == '\r' {
			hasEscapesOrNewlines = true
			break
		}
		end++
	}

	if !hasEscapesOrNewlines && end < len(l.input) && l.input[end] == quote {
		str := l.input[l.pos:end]
		l.column += utf8.RuneCountInString(str)
		l.pos = end
		l.readPos = end + 1
		l.char = rune(quote)
		l.size = 1
		return str
	}

	var out strings.Builder
	for l.char != rune(quote) && l.char != 0 {
		if l.char == '\\' {
			l.readChar()
			switch l.char {
			case 'n':
				out.WriteRune('\n')
			case 't':
				out.WriteRune('\t')
			case 'r':
				out.WriteRune('\r')
			case rune(quote):
				out.WriteRune(rune(quote))
			case '\\':
				out.WriteRune('\\')
			default:
				out.WriteRune('\\')
				out.WriteRune(l.char)
			}
		} else {
			out.WriteRune(l.char)
		}
		l.readChar()
	}
	return out.String()
}

// stripUnderscores removes PHP 7.4+ numeric separator underscores.
// Returns the original string unchanged when no underscores are present (no allocation).
func stripUnderscores(s string) string {
	if !strings.ContainsRune(s, '_') {
		return s
	}
	return strings.ReplaceAll(s, "_", "")
}

// readOctalNumber reads and processes an octal number (0o format)
func (l *Lexer) readOctalNumber() (string, bool) {
	l.readChar() // consume '0'
	l.readChar() // consume 'o' or 'O'
	start := l.pos
	for (l.char >= '0' && l.char <= '7') || l.char == '_' {
		l.readChar()
	}
	return "0o" + stripUnderscores(l.input[start:l.pos]), false
}

func (l *Lexer) readNumber() (string, bool) {
	position := l.pos
	isFloat := false

	// PHP 8 octal literal: 0o or 0O
	if l.char == '0' {
		switch l.peekChar() {
		case 'o', 'O':
			return l.readOctalNumber()
		case 'x', 'X':
			// Hexadecimal literal
			l.readChar() // consume '0'
			l.readChar() // consume 'x' or 'X'
			start := l.pos
			for (l.char >= '0' && l.char <= '9') || (l.char >= 'a' && l.char <= 'f') || (l.char >= 'A' && l.char <= 'F') || l.char == '_' {
				l.readChar()
			}
			return "0x" + stripUnderscores(l.input[start:l.pos]), false
		}
	}

	for isDigit(l.char) || l.char == '.' || l.char == '_' {
		if l.char == '.' {
			if isFloat { // Second decimal point
				break
			}
			isFloat = true
		}
		if l.char == '_' {
			// PHP 7.4+ numeric literal separator, skip underscore
			l.readChar()
			continue
		}
		l.readChar()
	}

	// Scientific notation exponent: e.g. "1e10", "1.2e+3", "1.7E-308".
	if l.char == 'e' || l.char == 'E' {
		lookaheadPos := l.readPos
		if lookaheadPos < len(l.input) && (l.input[lookaheadPos] == '+' || l.input[lookaheadPos] == '-') {
			lookaheadPos++
		}
		if lookaheadPos < len(l.input) && isDigit(rune(l.input[lookaheadPos])) {
			isFloat = true
			l.readChar() // consume 'e'/'E'
			if l.char == '+' || l.char == '-' {
				l.readChar()
			}
			for isDigit(l.char) {
				l.readChar()
			}
		}
	}

	return stripUnderscores(l.input[position:l.pos]), isFloat
}

// readIdentifier reads a PHP identifier (supports Unicode)
func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) NextToken() token.Token {
	if l.hasPeeked {
		tok := l.peekedToken
		l.hasPeeked = false
		return tok
	}
	return l.scanToken()
}

func (l *Lexer) scanToken() token.Token {
	if len(l.heredocTokens) > 0 {
		return l.nextHeredocToken()
	}
	if l.inHTML {
		return l.lexInlineHTML()
	}
	l.skipWhitespace()
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}

	// Attributes
	if l.char == '#' && l.peekChar() == '[' {
		return l.lexAttribute(pos)
	}
	if l.char == '#' {
		comment := l.readHashComment()
		return token.Token{Type: token.T_COMMENT, Literal: comment, Pos: pos}
	}

	switch l.char {
	case '?':
		return l.lexQuestion(pos)
	case 0:
		return token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	case '+', '-', '*', '/', '%', '|', '^', '>', '<', '$', '=', '(', ')', '{', '}', ';', ',', '&', '.', '"', '\'', '\\', ':', '[', ']', '!', '@', '~':
		return l.lexSymbol(pos)
	}

	if isLetter(l.char) {
		return l.lexIdentifier(pos)
	}
	if isDigit(l.char) {
		return l.lexNumber(pos)
	}

	tok := token.Token{Type: token.T_ILLEGAL, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

// atOpenTag reports whether the lexer is positioned at the start of a
// recognized PHP open tag ("<?php" or "<?="). Short open tags ("<?") are
// intentionally not treated as PHP open tags here, since they are disabled
// by default in modern PHP and commonly appear as literal text (e.g. XML
// declarations like "<?xml version=\"1.0\"?>") inside inline HTML.
func (l *Lexer) atOpenTag() bool {
	rest := l.input[l.pos:]
	if len(rest) >= 5 && strings.EqualFold(rest[:5], "<?php") {
		return true
	}
	return strings.HasPrefix(rest, "<?=")
}

// lexInlineHTML scans literal (non-PHP) content until the next recognized
// PHP open tag or EOF, emitting it as a single T_INLINE_HTML token. If the
// lexer is already positioned at an open tag, it delegates to lexOpenTag.
func (l *Lexer) lexInlineHTML() token.Token {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	if l.char == 0 {
		return token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	}
	start := l.pos
	for l.char != 0 {
		if l.char == '<' && l.atOpenTag() {
			break
		}
		l.readChar()
	}
	if l.pos > start {
		return token.Token{Type: token.T_INLINE_HTML, Literal: l.input[start:l.pos], Pos: pos}
	}
	return l.lexOpenTag(pos)
}

// lexOpenTag consumes a PHP open tag ("<?php" or "<?=") and switches the
// lexer into PHP-tokenization mode. "<?=" is shorthand for "<?php echo ",
// so it is expanded into an implicit T_ECHO token queued right behind the
// open tag.
func (l *Lexer) lexOpenTag(pos token.Position) token.Token {
	rest := l.input[l.pos:]
	switch {
	case len(rest) >= 5 && strings.EqualFold(rest[:5], "<?php"):
		for i := 0; i < 5; i++ {
			l.readChar()
		}
		l.inHTML = false
		return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
	case strings.HasPrefix(rest, "<?="):
		for i := 0; i < 3; i++ {
			l.readChar()
		}
		l.inHTML = false
		l.heredocTokens = []token.Token{{Type: token.T_ECHO, Literal: "echo", Pos: pos}}
		return token.Token{Type: token.T_OPEN_TAG, Literal: "<?=", Pos: pos}
	default:
		// Should be unreachable: callers only invoke lexOpenTag after
		// atOpenTag() confirmed a match. Fall back to a single literal
		// char to guarantee forward progress.
		l.readChar()
		return token.Token{Type: token.T_INLINE_HTML, Literal: "<", Pos: pos}
	}
}

// --- Helper methods for NextToken ---

func (l *Lexer) lexAttribute(pos token.Position) token.Token {
	startPos := l.pos
	startLine := l.line
	startCol := l.column
	l.readChar() // '#'
	l.readChar() // '['
	depth := 1
	for l.char != 0 && depth > 0 {
		// Skip string literals so '[' or ']' inside them don't affect depth.
		if l.char == '\'' || l.char == '"' {
			quote := l.char
			l.readChar() // consume opening quote
			for l.char != 0 && l.char != quote {
				if l.char == '\\' {
					l.readChar() // skip escaped char
				}
				l.readChar()
			}
			if l.char == quote {
				l.readChar() // consume closing quote
			}
			continue
		}
		if l.char == '[' {
			depth++
		} else if l.char == ']' {
			depth--
		}
		l.readChar()
	}
	endPos := l.pos
	attrLiteral := l.input[startPos:endPos]
	return token.Token{Type: token.T_ATTRIBUTE, Literal: attrLiteral, Pos: token.Position{Line: startLine, Column: startCol, Offset: startPos}}
}

func (l *Lexer) lexQuestion(pos token.Position) token.Token {
	if l.peekChar() == '-' && l.readPos+1 < len(l.input) && l.input[l.readPos+1] == '>' {
		l.readChar()
		l.readChar()
		l.readChar()
		return token.Token{Type: token.T_NULLSAFE_OBJECT_OPERATOR, Literal: "?->", Pos: pos}
	}
	if l.peekChar() == '>' {
		l.readChar() // consume '?', l.char now '>'
		l.readChar() // consume '>'
		// PHP consumes a single newline immediately following the close
		// tag so that e.g. "?>\n" doesn't emit a spurious blank HTML line.
		if l.char == '\r' && l.peekChar() == '\n' {
			l.readChar()
			l.readChar()
		} else if l.char == '\n' {
			l.readChar()
		}
		l.inHTML = true
		return token.Token{Type: token.T_CLOSE_TAG, Literal: "?>", Pos: pos}
	}
	if l.peekChar() == '?' {
		l.readChar()
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.T_COALESCE_EQUAL, Literal: "??=", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.T_COALESCE, Literal: "??", Pos: pos}
	}
	tok := token.Token{Type: token.T_QUESTION, Literal: asciiString(l.char), Pos: pos}
	l.readChar()
	return tok
}

func (l *Lexer) lexSymbol(pos token.Position) token.Token {
	// Implementation moved to lexer_symbol.go
	// This stub is left for reference; see lexer_symbol.go for helpers.
	switch l.char {
	case '+':
		return l.lexPlus(pos)
	case '-':
		return l.lexMinus(pos)
	case '*':
		return l.lexAsterisk(pos)
	case '/':
		return l.lexSlash(pos)
	case '%':
		return l.lexPercent(pos)
	case '|':
		return l.lexPipe(pos)
	case '^':
		return l.lexCaret(pos)
	case '>':
		return l.lexGreater(pos)
	case '<':
		return l.lexLess(pos)
	case '$':
		return l.lexDollar(pos)
	case '=':
		return l.lexEquals(pos)
	case '(': // ...single char tokens...
		return l.lexSingleChar(token.T_LPAREN, pos)
	case ')':
		return l.lexSingleChar(token.T_RPAREN, pos)
	case '{':
		return l.lexSingleChar(token.T_LBRACE, pos)
	case '}':
		return l.lexSingleChar(token.T_RBRACE, pos)
	case ';':
		return l.lexSingleChar(token.T_SEMICOLON, pos)
	case ',':
		return l.lexSingleChar(token.T_COMMA, pos)
	case '&':
		return l.lexAmpersand(pos)
	case '.':
		if isDigit(l.peekChar()) {
			return l.lexNumber(pos)
		}
		return l.lexDot(pos)
	case '"':
		return l.lexDoubleQuote(pos)
	case '\\':
		return l.lexBackslash(pos)
	case '\'':
		return l.lexSingleQuote(pos)
	case ':':
		return l.lexColon(pos)
	case '[':
		return l.lexSingleChar(token.T_LBRACKET, pos)
	case ']':
		return l.lexSingleChar(token.T_RBRACKET, pos)
	case '!':
		return l.lexBang(pos)
	case '@':
		return l.lexSingleChar(token.T_AT, pos)
	case '~':
		return l.lexSingleChar(token.T_TILDE, pos)
	}
	return token.Token{Type: token.T_ILLEGAL, Literal: asciiString(l.char), Pos: pos}
}

func (l *Lexer) lexIdentifier(pos token.Position) token.Token {
	ident := l.readIdentifier()
	return LookupKeyword(ident, pos)
}

func (l *Lexer) lexNumber(pos token.Position) token.Token {
	num, isFloat := l.readNumber()
	if isFloat {
		return token.Token{Type: token.T_DNUMBER, Literal: num, Pos: pos}
	}
	return token.Token{Type: token.T_LNUMBER, Literal: num, Pos: pos}
}
