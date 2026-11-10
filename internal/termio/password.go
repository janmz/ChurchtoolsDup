// Package termio reads interactive terminal input.
package termio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// MaxLineBytes limits one line of interactive input (passwords, tokens,
// menu choices) so piped input cannot grow memory without bound.
const MaxLineBytes = 4096

// ErrLineTooLong is returned when an input line exceeds MaxLineBytes.
var ErrLineTooLong = errors.New("eingabezeile zu lang")

// ReadLine reads one line from r without the line break. A last line without
// line break is accepted; io.EOF is only returned when nothing was read.
func ReadLine(r *bufio.Reader, max int) (string, error) {
	var line []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(line) > 0 {
				break
			}
			return "", err
		}
		if b == '\n' {
			break
		}
		if len(line) >= max {
			return "", ErrLineTooLong
		}
		line = append(line, b)
	}
	return strings.TrimSuffix(string(line), "\r"), nil
}

// ReadPassword prints prompt to stderr and reads a password without echo (no
// asterisks, so the length is not revealed). UTF-8 input such as umlauts is
// kept. Without a terminal the password is read as one line from in, which
// must be the same buffered reader used for all other stdin input so no
// buffered line is lost. The password is returned verbatim, including
// leading or trailing spaces.
func ReadPassword(prompt string, in *bufio.Reader) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		password, err := readPasswordPlain(in)
		fmt.Fprintln(os.Stderr)
		return password, err
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		password, readErr := readPasswordPlain(in)
		fmt.Fprintln(os.Stderr)
		return password, readErr
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	password, err := readPasswordRaw(os.Stdin)
	fmt.Fprint(os.Stderr, "\r\n")
	return password, err
}

// readPasswordRaw reads bytes from a terminal in raw mode until Enter. It
// collects complete UTF-8 sequences and handles backspace per character;
// nothing is echoed. Ctrl+C / Ctrl+D abort.
func readPasswordRaw(r io.Reader) (string, error) {
	var password []byte
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) && len(password) > 0 {
				return string(password), nil
			}
			return "", err
		}
		if n == 0 {
			continue
		}

		b := buf[0]
		switch {
		case b == '\r' || b == '\n':
			return string(password), nil
		case b == 3 || b == 4:
			return "", io.EOF
		case b == 8 || b == 127:
			if len(password) > 0 {
				_, size := utf8.DecodeLastRune(password)
				password = password[:len(password)-size]
			}
		case b < 32:
			// other control characters are ignored
		default:
			if len(password) >= MaxLineBytes {
				return "", ErrLineTooLong
			}
			password = append(password, b)
		}
	}
}

// readPasswordPlain reads one line; a last line without line break is
// accepted. Only the line break is removed.
func readPasswordPlain(r io.Reader) (string, error) {
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	return ReadLine(reader, MaxLineBytes)
}
