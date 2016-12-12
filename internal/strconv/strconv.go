package strconv

import (
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrSyntax = strconv.ErrSyntax
	ErrRange  = strconv.ErrRange
)

func Atoi(s string) (i int, err error) {
	i64, err := ParseInt(s)
	return int(i64), err
}

func Itoa(i int) string {
	return FormatInt(int64(i), 10)
}

func AppendInt(dst []byte, i int64, base int) []byte {
	return strconv.AppendInt(dst, i, base)
}

func FormatInt(i int64, base int) string {
	return strconv.FormatInt(i, base)
}

func FormatUint(u uint64, base int) string {
	return strconv.FormatUint(u, base)
}

func FormatFloat(f float64, fmt byte, prec, bitSize int) string {
	s := strconv.FormatFloat(f, fmt, prec, bitSize)

	switch s {
	case "NaN":
		return "nan"
	case "-Inf":
		return "-inf"
	case "+Inf":
		return "inf"
	}

	return s
}

func ParseUint(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}

	if s[0] == '0' && len(s) != 1 && (s[1] == 'x' || s[1] == 'X') {
		u, err := strconv.ParseUint(s[2:], 16, 64)
		return u, unwrap(err)
	}

	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		f, err := strconv.ParseFloat(s, 64)
		return uint64(f), unwrap(err)
	}

	return i, nil
}

func ParseInt(s string) (int64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}

	if s[0] == '-' {
		if len(s) > 2 && s[1] == '0' && (s[2] == 'x' || s[2] == 'X') {
			i, err := strconv.ParseInt("-"+s[3:], 16, 64)
			return i, unwrap(err)
		}
	} else {
		if len(s) > 1 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
			i, err := strconv.ParseInt(s[2:], 16, 64)
			return i, unwrap(err)
		}
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return i, unwrap(err)
	}

	return i, nil
}

func ParseFloat(s string) (float64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}

	var f float64
	var err error
	{
		if s[0] == '-' {
			if len(s) > 2 && s[1] == '0' && (s[2] == 'x' || s[2] == 'X') {
				f, err = parseHexFloat("-" + s[3:])

				goto parseEnd
			}
		} else {
			if len(s) > 1 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
				f, err = parseHexFloat(s[2:])

				goto parseEnd
			}
		}

		f, err = strconv.ParseFloat(s, 64)
		if err != nil {
			err = unwrap(err)
		}
	}

parseEnd:
	if err != nil {
		return 0, err
	}

	if f == 0 {
		if s[0] == '-' {
			f = math.Copysign(0, -1)
		}
	}

	return f, nil
}

func ScanUint(sc io.ByteScanner) (uint64, error) {
	s := newScanner(sc)

	s.next()

	u64, err := s.scanUint()
	if err != nil {
		return 0, err
	}

	err = s.sc.UnreadByte()
	if err != nil {
		return 0, err
	}

	return u64, err
}

func ScanInt(sc io.ByteScanner) (int64, error) {
	s := newScanner(sc)

	s.next()

	i64, err := s.scanInt()
	if err != nil {
		return 0, err
	}

	err = s.sc.UnreadByte()
	if err != nil {
		return 0, err
	}

	return i64, err
}

func ScanFloat(sc io.ByteScanner) (float64, error) {
	s := newScanner(sc)

	s.next()

	f64, err := s.scanFloat()
	if err != nil {
		return 0, err
	}

	err = s.sc.UnreadByte()
	if err != nil {
		return 0, err
	}

	return f64, err
}

func parseHexFloat(s string) (float64, error) {
	if len(s) == 0 {
		return 0, ErrSyntax
	}

	var neg bool

	if s[0] == '-' {
		neg = true
		s = s[1:]
	}

	var integer string
	var fraction string
	var exponent string

	if j := strings.IndexRune(s, '.'); j != -1 {
		integer = s[:j]
		s = s[j+1:]
		if k := strings.IndexAny(s, "pP"); k != -1 {
			fraction = s[:k]
			exponent = s[k+1:]
		} else {
			fraction = s
		}
	} else {
		if k := strings.IndexAny(s, "pP"); k != -1 {
			integer = s[:k]
			exponent = s[k+1:]
		} else {
			integer = s
		}
	}

	var f float64

	if integer != "" {
		i, err := strconv.ParseInt(integer, 16, 64)
		if err != nil {
			return 0, unwrap(err)
		}

		f = float64(i)
	}

	if fraction != "" {
		coef := 16.0

		var x int
		for _, r := range fraction {
			if r >= utf8.RuneSelf {
				return 0, ErrSyntax
			}
			x = digitVal(byte(r))
			if x == 16 {
				return 0, ErrSyntax
			}

			// do nothing
			if x == '0' {
				coef *= 16
				continue
			}

			f += float64(x) / coef

			coef *= 16
		}
	}

	if exponent != "" {
		e, err := strconv.ParseInt(exponent, 10, 64)
		if err != nil {
			return 0, unwrap(err)
		}

		f = f * math.Pow(2, float64(e))
	}

	if neg {
		f = -f
	}

	return f, nil
}

func digitVal(c byte) int {
	switch {
	case uint(c)-'0' < 10:
		return int(c - '0')
	case uint(c)-'a' < 6:
		return int(c - 'a' + 10)
	case uint(c)-'A' < 6:
		return int(c - 'A' + 10)
	}

	return 16
}

func unwrap(err error) error {
	if err == nil {
		return nil
	}

	return err.(*strconv.NumError).Err
}
