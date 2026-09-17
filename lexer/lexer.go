package lexer

import (
	"bytes"
	"context"
	"strings"
	"unicode/utf8"

	"github.com/ayanozturk/go-php-parser/token"
)

var (
	openTagPHP  = []byte("<?php")
	openTagEcho = []byte("<?=")
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
	input    []byte
	lines    token.LineTable
	pos      int
	readPos  int
	char     rune // Unicode-aware current character
	size     int  // Size of last rune read
	line     int
	column   int
	inString bool // Tracks if currently inside a string
	// Pending significant tokens (heredoc/encapsed/<?= echo expansion).
	heredocTokens []token.Token
	// Lookahead cache: avoids state save/restore on PeekToken
	hasPeeked   bool
	peekedToken token.Token
	// inHTML tracks whether the lexer is currently scanning literal
	// (non-PHP) content outside of <?php ... ?> tags. PHP files may start
	// with, contain, or end with inline HTML/text; this flag switches the
	// scanner between raw-text mode and normal PHP tokenization.
	inHTML bool

	// Encapsed / heredoc interpolation state (Zend-aligned).
	encapsed       encapsedMode
	heredocLabel   string
	heredocNowdoc  bool
	braceExprDepth int
	afterCurlyOpen bool
	// lastSignificant is the previous non-trivia token type, used so
	// semi-reserved words (enum, ...) stay T_STRING after \\ / class / etc.
	lastSignificant token.TokenType
}

// inStringMode returns whether the lexer is currently inside a string.
func (l *Lexer) inStringMode() bool {
	return l.inString
}

// State is an opaque snapshot of the lexer's scanning position, usable with
// Snapshot/Restore to support small bounded lookahead where a single
// peeked token isn't enough to disambiguate a construct (e.g. distinguishing
// "public(set)" from a property type that happens to start with "(").
type State struct {
	pos, readPos    int
	char            rune
	size            int
	line, column    int
	inString        bool
	heredocTokens   []token.Token
	hasPeeked       bool
	peekedToken     token.Token
	inHTML          bool
	encapsed        encapsedMode
	heredocLabel    string
	heredocNowdoc   bool
	braceExprDepth  int
	afterCurlyOpen  bool
	lastSignificant token.TokenType
}

// Snapshot captures the current lexer position so it can be restored later.
func (l *Lexer) Snapshot() State {
	return State{
		pos:             l.pos,
		readPos:         l.readPos,
		char:            l.char,
		size:            l.size,
		line:            l.line,
		column:          l.column,
		inString:        l.inString,
		heredocTokens:   append([]token.Token(nil), l.heredocTokens...),
		hasPeeked:       l.hasPeeked,
		peekedToken:     l.peekedToken,
		inHTML:          l.inHTML,
		encapsed:        l.encapsed,
		heredocLabel:    l.heredocLabel,
		heredocNowdoc:   l.heredocNowdoc,
		braceExprDepth:  l.braceExprDepth,
		afterCurlyOpen:  l.afterCurlyOpen,
		lastSignificant: l.lastSignificant,
	}
}

// Restore rewinds the lexer to a previously captured Snapshot.
func (l *Lexer) Restore(s State) {
	l.pos = s.pos
	l.readPos = s.readPos
	l.char = s.char
	l.size = s.size
	l.line = s.line
	l.column = s.column
	l.inString = s.inString
	l.heredocTokens = append([]token.Token(nil), s.heredocTokens...)
	l.hasPeeked = s.hasPeeked
	l.peekedToken = s.peekedToken
	l.inHTML = s.inHTML
	l.encapsed = s.encapsed
	l.heredocLabel = s.heredocLabel
	l.heredocNowdoc = s.heredocNowdoc
	l.braceExprDepth = s.braceExprDepth
	l.afterCurlyOpen = s.afterCurlyOpen
	l.lastSignificant = s.lastSignificant
}

func New(input string) *Lexer {
	return NewBytes([]byte(input))
}

func NewBytes(input []byte) *Lexer {
	l := &Lexer{
		input:  input,
		lines:  token.NewLineTable(input),
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// NewFile constructs a Lexer for a real PHP source file (as opposed to a
// bare code snippet used in tests), starting in inline-HTML mode unless the
// file begins with a recognized PHP open tag. PHP allows a file to start
// with arbitrary literal content before its first "<?php"/"<?=" tag (e.g.
// template/view files); New always starts in PHP-code mode instead, since
// many callers construct bare snippets (operators, expressions) with no
// leading open tag and expect immediate PHP tokenization.
func NewFile(input string) *Lexer {
	return NewFileBytes([]byte(input))
}

func NewFileBytes(input []byte) *Lexer {
	l := NewBytes(input)
	if !l.atOpenTag() {
		l.inHTML = true
	}
	return l
}

func (l *Lexer) Source() []byte {
	return l.input
}

func (l *Lexer) LineTable() token.LineTable {
	return l.lines
}

func (l *Lexer) text(start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(l.input) {
		end = len(l.input)
	}
	if start >= end {
		return ""
	}
	return string(l.input[start:end])
}

func (l *Lexer) finishToken(tok token.Token) token.Token {
	if tok.End.Line != 0 || tok.End.Column != 0 || tok.End.Offset != 0 {
		return tok
	}
	tok.End = token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	return tok
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
			l.char, l.size = utf8.DecodeRune(l.input[l.readPos:])
		}
	}
	l.pos = l.readPos
	l.readPos += l.size
}

// atEOF reports whether the lexer has consumed the entire input. This must
// be used instead of comparing l.char == 0 to detect end-of-input, because a
// literal NUL byte inside a PHP string (valid, if rare, source content) also
// makes l.char == 0 while l.pos is still a valid in-bounds offset.
func (l *Lexer) atEOF() bool {
	return l.pos >= len(l.input)
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
	r, _ := utf8.DecodeRune(l.input[l.readPos:])
	return r
}

// SkipBalancedCurlyBlock advances from the current "{" through its matching
// "}". It is kept as the compatibility API for callers that do not need the
// closing position.
func (l *Lexer) SkipBalancedCurlyBlock() bool {
	_, ok := l.SkipBalancedCurlyBlockWithEnd()
	return ok
}

// SkipBalancedCurlyBlockWithEnd also returns the position immediately after
// the closing brace for declaration-span construction.
func (l *Lexer) SkipBalancedCurlyBlockWithEnd() (token.Position, bool) {
	depth := 0
	if l.char != '{' {
		depth = 1
	}
	for !l.atEOF() {
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
				return token.Position{Line: l.line, Column: l.column, Offset: l.pos}, true
			}
			continue
		}
		l.readChar()
	}
	return token.Position{Line: l.line, Column: l.column, Offset: l.pos}, false
}

func (l *Lexer) skipQuotedString(quote rune) {
	l.readChar()
	for !l.atEOF() {
		if l.char == '\\' {
			l.readChar()
			if !l.atEOF() {
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
	for !l.atEOF() && l.char != '\n' {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar()
	l.readChar()
	for !l.atEOF() {
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
		str := l.text(l.pos, end)
		l.column += utf8.RuneCountInString(str)
		l.pos = end
		l.readPos = end + 1
		l.char = rune(quote)
		l.size = 1
		return str
	}

	var out strings.Builder
	for l.char != rune(quote) && !l.atEOF() {
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

func (l *Lexer) readNumber() (string, bool) {
	position := l.pos
	isFloat := false

	// PHP 8 octal literal: 0o or 0O
	if l.char == '0' {
		switch l.peekChar() {
		case 'o', 'O':
			return l.readOctalNumber()
		case 'b', 'B':
			// Binary literal: e.g. "0b1010".
			l.readChar() // consume '0'
			l.readChar() // consume 'b' or 'B'
			for l.char == '0' || l.char == '1' || l.char == '_' {
				l.readChar()
			}
			return l.text(position, l.pos), false
		case 'x', 'X':
			// Hexadecimal literal
			l.readChar() // consume '0'
			l.readChar() // consume 'x' or 'X'
			for (l.char >= '0' && l.char <= '9') || (l.char >= 'a' && l.char <= 'f') || (l.char >= 'A' && l.char <= 'F') || l.char == '_' {
				l.readChar()
			}
			return l.text(position, l.pos), false
		}
	}

	for isDigit(l.char) || l.char == '.' || l.char == '_' {
		if l.char == '.' {
			if isFloat { // Second decimal point
				break
			}
			isFloat = true
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

	return l.text(position, l.pos), isFloat
}

func (l *Lexer) readOctalNumber() (string, bool) {
	position := l.pos
	l.readChar() // consume '0'
	l.readChar() // consume 'o' or 'O'
	for (l.char >= '0' && l.char <= '7') || l.char == '_' {
		l.readChar()
	}
	return l.text(position, l.pos), false
}

// readIdentifier reads a PHP identifier (supports Unicode)
func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.char) || isDigit(l.char) {
		l.readChar()
	}
	return l.text(start, l.pos)
}

func (l *Lexer) NextToken() token.Token {
	if l.hasPeeked {
		tok := l.peekedToken
		l.hasPeeked = false
		if tok.Type != token.T_EOF {
			l.lastSignificant = tok.Type
		}
		return tok
	}
	tok := l.finishToken(l.scanToken())
	if tok.Type != token.T_EOF {
		l.lastSignificant = tok.Type
	}
	return tok
}

func (l *Lexer) scanToken() token.Token {
	if len(l.heredocTokens) > 0 {
		return l.nextHeredocToken()
	}
	if l.encapsed != encapsedNone {
		l.queueEncapsedBody(l.heredocNowdoc)
		if len(l.heredocTokens) > 0 {
			return l.nextHeredocToken()
		}
	}
	if l.inHTML {
		return l.lexInlineHTML()
	}

	trivia := l.collectLeadingTrivia()
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}

	if l.atEOF() {
		tok := token.Token{Type: token.T_EOF, Literal: "", Pos: pos, End: pos}
		tok.TrailingTrivia = trivia
		return tok
	}

	// Attributes: T_ATTRIBUTE is only "#[" (Zend).
	if l.char == '#' && l.peekChar() == '[' {
		tok := l.lexAttribute(pos)
		tok.LeadingTrivia = trivia
		return tok
	}

	var tok token.Token
	switch l.char {
	case '?':
		tok = l.lexQuestion(pos)
	case '+', '-', '*', '/', '%', '|', '^', '>', '<', '$', '=', '(', ')', '{', '}', ';', ',', '&', '.', '"', '\'', '\\', ':', '[', ']', '!', '@', '~':
		tok = l.lexSymbol(pos)
	default:
		if isLetter(l.char) {
			tok = l.lexIdentifier(pos)
		} else if isDigit(l.char) {
			tok = l.lexNumber(pos)
		} else {
			tok = token.Token{Type: token.T_ILLEGAL, Literal: asciiString(l.char), Pos: pos}
			l.readChar()
			tok.End = token.Position{Line: l.line, Column: l.column, Offset: l.pos}
		}
	}

	tok.LeadingTrivia = trivia
	// After {$expr} / ${...}, a closing '}' resumes encapsed scanning.
	if l.afterCurlyOpen {
		switch tok.Type {
		case token.T_LBRACE:
			l.braceExprDepth++
		case token.T_RBRACE:
			l.braceExprDepth--
			if l.braceExprDepth <= 0 {
				// finishToken fills End from l.pos when unset. resumeEncapsedAfterBrace
				// advances the lexer to queue the rest of the encapsed/heredoc body,
				// so capture End for '}' before that scan or Width() swallows body bytes.
				if tok.End.Line == 0 && tok.End.Column == 0 && tok.End.Offset == 0 {
					tok.End = token.Position{Line: l.line, Column: l.column, Offset: l.pos}
				}
				l.resumeEncapsedAfterBrace()
			}
		}
	}
	return tok
}

// atOpenTag reports whether the lexer is positioned at the start of a
// recognized PHP open tag ("<?php" or "<?="). Short open tags ("<?") are
// intentionally not treated as PHP open tags here, since they are disabled
// by default in modern PHP and commonly appear as literal text (e.g. XML
// declarations like "<?xml version=\"1.0\"?>") inside inline HTML.
func (l *Lexer) atOpenTag() bool {
	rest := l.input[l.pos:]
	if len(rest) >= 5 && bytes.EqualFold(rest[:5], openTagPHP) {
		return true
	}
	return bytes.HasPrefix(rest, openTagEcho)
}

// lexInlineHTML scans literal (non-PHP) content until the next recognized
// PHP open tag or EOF, emitting it as a single T_INLINE_HTML token. If the
// lexer is already positioned at an open tag, it delegates to lexOpenTag.
func (l *Lexer) lexInlineHTML() token.Token {
	pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	if l.atEOF() {
		return token.Token{Type: token.T_EOF, Literal: "", Pos: pos}
	}
	start := l.pos
	for !l.atEOF() {
		if l.char == '<' && l.atOpenTag() {
			break
		}
		l.readChar()
	}
	if l.pos > start {
		return token.Token{Type: token.T_INLINE_HTML, Literal: l.text(start, l.pos), Pos: pos}
	}
	return l.lexOpenTag(pos)
}

// lexOpenTag consumes a PHP open tag ("<?php" or "<?=") and switches the
// lexer into PHP-tokenization mode. "<?=" is T_OPEN_TAG_WITH_ECHO (Zend),
// covering exactly those three source bytes — never a synthetic T_ECHO
// that would inflate green widths past the file length.
func (l *Lexer) lexOpenTag(pos token.Position) token.Token {
	rest := l.input[l.pos:]
	switch {
	case len(rest) >= 5 && bytes.EqualFold(rest[:5], openTagPHP):
		for i := 0; i < 5; i++ {
			l.readChar()
		}
		l.inHTML = false
		return token.Token{Type: token.T_OPEN_TAG, Literal: "<?php", Pos: pos}
	case bytes.HasPrefix(rest, openTagEcho):
		for i := 0; i < 3; i++ {
			l.readChar()
		}
		l.inHTML = false
		return token.Token{Type: token.T_OPEN_TAG_WITH_ECHO, Literal: "<?=", Pos: pos}
	default:
		// Should be unreachable: callers only invoke lexOpenTag after
		// atOpenTag() confirmed a match. Fall back to a single literal
		// char to guarantee forward progress.
		l.readChar()
		return token.Token{Type: token.T_INLINE_HTML, Literal: "<", Pos: pos}
	}
}

// --- Helper methods for NextToken ---

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
	tok := LookupKeyword(ident, pos)
	tok.End = token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	// Semi-reserved keywords are T_STRING in name contexts (App\Enum, class Enum).
	if isSemiReserved(tok.Type) && forcesStringIdent(l.lastSignificant) {
		tok.Type = token.T_STRING
	}
	return tok
}

func forcesStringIdent(prev token.TokenType) bool {
	switch prev {
	case token.T_NS_SEPARATOR, token.T_CLASS, token.T_INTERFACE, token.T_TRAIT, token.T_ENUM,
		token.T_FUNCTION, token.T_CONST, token.T_EXTENDS, token.T_IMPLEMENTS,
		token.T_AS, token.T_OBJECT_OPERATOR, token.T_NULLSAFE_OBJECT_OPERATOR,
		token.T_DOUBLE_COLON, token.T_GOTO, token.T_NAMESPACE:
		return true
	default:
		return false
	}
}

func isSemiReserved(tt token.TokenType) bool {
	switch tt {
	case token.T_ENUM, token.T_MATCH, token.T_SELF, token.T_PARENT, token.T_STATIC,
		token.T_MIXED, token.T_NEVER, token.T_TRUE, token.T_FALSE, token.T_NULL,
		token.T_ARRAY, token.T_CALLABLE, token.T_READONLY,
		token.T_ABSTRACT, token.T_FINAL, token.T_PUBLIC, token.T_PRIVATE, token.T_PROTECTED,
		token.T_LOGICAL_AND, token.T_LOGICAL_OR, token.T_LOGICAL_XOR:
		return true
	default:
		return false
	}
}

func (l *Lexer) lexNumber(pos token.Position) token.Token {
	num, isFloat := l.readNumber()
	end := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
	if isFloat {
		return token.Token{Type: token.T_DNUMBER, Literal: num, Pos: pos, End: end}
	}
	return token.Token{Type: token.T_LNUMBER, Literal: num, Pos: pos, End: end}
}

// LexAll returns every significant token with trivia attached, through T_EOF.
func LexAll(src []byte) []token.Token {
	toks, _ := LexAllContext(nil, src)
	return toks
}

// LexAllContext returns every significant token with trivia attached, through
// T_EOF, while allowing callers to stop between tokens. A nil context disables
// cancellation checks and preserves LexAll's existing behavior.
func LexAllContext(ctx context.Context, src []byte) ([]token.Token, error) {
	l := NewFileBytes(src)
	var toks []token.Token
	for {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return toks, err
			}
		}
		tok := l.NextToken()
		toks = append(toks, tok)
		if tok.Type == token.T_EOF {
			return toks, nil
		}
	}
}

// PrintTokens concatenates leading trivia + token text (+ trailing on EOF)
// covering the full source. Identity: PrintTokens(LexAll(src), src) == string(src)
// when the lexer is lossless.
func PrintTokens(toks []token.Token, src []byte) string {
	var b strings.Builder
	b.Grow(len(src))
	for _, tok := range toks {
		for _, tr := range tok.LeadingTrivia {
			b.WriteString(tr.Text(src))
		}
		if tok.Type != token.T_EOF {
			b.WriteString(tok.Text(src))
		}
		for _, tr := range tok.TrailingTrivia {
			b.WriteString(tr.Text(src))
		}
	}
	return b.String()
}
