package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota // error occurred
	itemEOF
	itemText
	itemComment
	itemLeftDelim
	itemRightDelim
	itemVariable
)

// item represents a token or text string returned from the scanner
type item struct {
	typ itemType // the type of this item
	pos Pos      // the starting position (in bytes) of this item in the input stream
	val string   // the value of this item
}

func (i item) String() string {
	switch {
	case i.typ == itemError:
		return i.val
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemComment:
		return fmt.Sprintf("<COMMENT - %q />", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

const eof = -1

type lexer struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	leftDelim  string    // start of action
	rightDelim string    // end of action
	state      stateFn   // the next lexing function to enter
	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	lastPos    Pos       // position of most recent item returned by nextItem
	items      chan item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs

}

func (l *lexer) String() string {
	return fmt.Sprintf("start: %v, pos: %v\n", l.start, l.pos)
}

type stateFn func(*lexer) stateFn

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

const defaultLeftDelim = "{{"
const defaultRightDelim = "}}"

// lex creates a new scanner for the input string.
func lex(name, input, left, right string) *lexer {
	if left == "" {
		left = defaultLeftDelim
	}
	if right == "" {
		right = defaultRightDelim
	}
	l := &lexer{
		name:       name,
		input:      input,
		leftDelim:  left,
		rightDelim: right,
		items:      make(chan item),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
	// Ensure that the consumer will stop iterating the channel
	close(l.items)
}

// lexText scans until an opening action delimiter, "{{".
func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {
			l.emitAnyText()
			return lexLeftDelim
		}
		if l.next() == eof {
			break
		}
	}

	// Correctly reached EOF.
	l.emitAnyText()
	l.emit(itemEOF)
	return nil
}

func (l *lexer) emitAnyText() {
	if l.pos > l.start {
		l.emit(itemText)
	}
}

// lexLeftDelim scans the left delimiter, which is known to be present.
func lexLeftDelim(l *lexer) stateFn {
	l.pos += Pos(len(l.leftDelim))
	s := l.input[l.pos:]
	switch {
	case strings.HasPrefix(s, "!"):
		return lexComment
	case strings.HasPrefix(s, "#"):
		return lexSection
	case strings.HasPrefix(s, "^"):
		return lexPartial
	case strings.HasPrefix(s, "{"):
		return lexRawText
	}
	l.emit(itemLeftDelim)
	// l.parenDepth = 0
	return lexInsideDelim
}

func lexInsideDelim(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {
			l.emit(itemVariable)
			return lexRightDelim
		}
		if l.next() == eof {
			break
		}
	}
	l.emitAnyText()
	l.emit(itemEOF)
	return nil
}

// lexComment scans a comment. The left comment marker is known to be present.
func lexComment(l *lexer) stateFn {
	// TODO(jabley): emit leftComment?
	l.pos += Pos(len("*"))
	l.ignore()

	i := strings.Index(l.input[l.pos:], l.rightDelim)
	if i < 0 {
		return l.errorf("unclosed comment")
	}

	l.pos += Pos(i)
	l.emit(itemComment)

	// TODO(jabley): emit rightComment?
	l.pos += Pos(len(l.rightDelim))
	l.ignore()

	return lexText
}

// lexRightDelim scans the right delimiter, which is known to be present.
func lexRightDelim(l *lexer) stateFn {
	l.pos += Pos(len(l.rightDelim))
	l.emit(itemRightDelim)
	return lexText
}

func lexSection(l *lexer) stateFn {
	return l.errorf("Section support not implemented")
}

func lexPartial(l *lexer) stateFn {
	return l.errorf("Partial support not implemented")
}

func lexRawText(l *lexer) stateFn {
	return l.errorf("Raw support not implemented")
}

func lexInterpolation(l *lexer) stateFn {
	return l.errorf("Interpolation support not implemented")
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
