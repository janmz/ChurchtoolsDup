package termio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ReadPassword reads a password from stdin and prints an asterisk per character
// on interactive terminals. Non-TTY input is read without echo.
func ReadPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, prompt)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		password, err := readPasswordPlain(os.Stdin)
		fmt.Fprintln(os.Stdout)
		return password, err
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		password, readErr := readPasswordPlain(os.Stdin)
		fmt.Fprintln(os.Stdout)
		return password, readErr
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	var password []rune
	buf := make([]byte, 1)
	for {
		n, readErr := os.Stdin.Read(buf)
		if readErr != nil {
			fmt.Fprintln(os.Stdout)
			if readErr == io.EOF && len(password) > 0 {
				return string(password), nil
			}
			return "", readErr
		}
		if n == 0 {
			continue
		}

		switch buf[0] {
		case '\r', '\n':
			fmt.Fprintln(os.Stdout)
			return string(password), nil
		case 3, 4:
			fmt.Fprintln(os.Stdout)
			return "", io.EOF
		case 8, 127:
			if len(password) > 0 {
				password = password[:len(password)-1]
				fmt.Fprint(os.Stdout, "\b \b")
			}
		default:
			if buf[0] >= 32 && buf[0] < 127 {
				password = append(password, rune(buf[0]))
				fmt.Fprint(os.Stdout, "*")
			}
		}
	}
}

func readPasswordPlain(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
