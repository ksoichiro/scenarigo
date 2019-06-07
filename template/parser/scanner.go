package parser

import (
	"bufio"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/zoncoen/scenarigo/template/token"
)

// eof represents invalid code points.
var eof = unicode.ReplacementChar

type scanner struct {
	r                  *bufio.Reader
	pos                int
	buf                []rune
	isReadingParameter bool
}

func newScanner(r io.Reader) *scanner {
	return &scanner{
		r:   bufio.NewReader(r),
		pos: 1,
	}
}

func (s *scanner) read() rune {
	if len(s.buf) > 0 {
		var ch rune
		ch, s.buf = s.buf[0], s.buf[1:]
		s.pos++
		return ch
	}
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	s.pos++
	return ch
}

func (s *scanner) unread(ch rune) {
	s.buf = append(s.buf, ch)
	s.pos--
}

func (s *scanner) skipSpaces() {
	for {
		if ch := s.read(); ch != ' ' {
			s.unread(ch)
			return
		}
	}
}

func (s *scanner) scanRawString() (int, token.Token, string) {
	var b strings.Builder
scan:
	for {
		switch ch := s.read(); ch {
		case eof:
			if b.Len() == 0 {
				return s.pos, token.EOF, ""
			}
			break scan
		case '{':
			next := s.read()
			if next == '{' {
				if b.Len() == 0 {
					return s.pos - 2, token.LDBRACE, "{{"
				}
				s.unread(ch)
				s.unread(next)
				break scan
			}
			s.unread(next)
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}
	return s.pos - b.Len(), token.STRING, b.String()
}

func (s *scanner) scanString() (int, token.Token, string) {
	var b strings.Builder
scan:
	for {
		ch := s.read()
		switch ch {
		case eof:
			// string not terminated
			return s.pos, token.ILLEGAL, ""
		case '"':
			break scan
		default:
			b.WriteRune(ch)
		}
	}
	return s.pos - b.Len() - 2, token.STRING, b.String()
}

func (s *scanner) scanInt(head rune) (int, token.Token, string) {
	var b strings.Builder
	b.WriteRune(head)
scan:
	for {
		ch := s.read()
		if !isDigit(ch) {
			s.unread(ch)
			break scan
		}
		b.WriteRune(ch)
	}
	if head == '0' && b.Len() != 1 {
		return s.pos - b.Len(), token.ILLEGAL, b.String()
	}
	return s.pos - b.Len(), token.INT, b.String()
}

func (s *scanner) scanIdent(head rune) (int, token.Token, string) {
	var b strings.Builder
	b.WriteRune(head)
scan:
	for {
		ch := s.read()
		switch ch {
		case '-', '_':
			b.WriteRune(ch)
			continue
		default:
			if isLetter(ch) || isDigit(ch) {
				b.WriteRune(ch)
				continue
			}
		}
		s.unread(ch)
		break scan
	}
	return s.pos - b.Len(), token.IDENT, b.String()
}

func (s *scanner) scan() (int, token.Token, string) {
	if !s.isReadingParameter {
		pos, tok, lit := s.scanRawString()
		if tok == token.LDBRACE {
			s.isReadingParameter = true
		}
		return pos, tok, lit
	}

	s.skipSpaces()
	ch := s.read()
	switch ch {
	case eof:
		return s.pos, token.EOF, ""
	case '(':
		return s.pos - 1, token.LPAREN, "("
	case ')':
		return s.pos - 1, token.RPAREN, ")"
	case '[':
		return s.pos - 1, token.LBRACK, "["
	case ']':
		return s.pos - 1, token.RBRACK, "]"
	case '}':
		next := s.read()
		if next == '}' {
			s.isReadingParameter = false
			return s.pos - 2, token.RDBRACE, "}}"
		}
		s.unread(next)
	case ',':
		return s.pos - 1, token.COMMA, ","
	case '.':
		return s.pos - 1, token.PERIOD, "."
	case '+':
		return s.pos - 1, token.ADD, "+"
	default:
		if ch == '"' {
			return s.scanString()
		}
		if isDigit(ch) {
			return s.scanInt(ch)
		}
		if isLetter(ch) {
			return s.scanIdent(ch)
		}
	}
	return s.pos - 1, token.ILLEGAL, string(ch)
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}
