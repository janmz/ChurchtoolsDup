// Package iolimit reads streams and files with an upper size bound so that a
// faulty server response or a wrongly chosen file cannot exhaust memory.
package iolimit

import (
	"fmt"
	"io"
	"os"
)

// Size limits used across the application.
const (
	MaxAPIResponseBytes int64 = 32 << 20  // 32 MiB per HTTP response
	MaxAPITotalBytes    int64 = 512 << 20 // 512 MiB for all pages of one list
	MaxCSVBytes         int64 = 100 << 20 // 100 MiB per CSV file
	MaxConfigBytes      int64 = 1 << 20   // 1 MiB per config file
	MaxErrorBodyBytes         = 2048      // bytes of a response body kept in error messages
)

// ErrTooLarge is wrapped by ReadAll and ReadFile when the limit is exceeded.
var ErrTooLarge = fmt.Errorf("daten zu groß")

// ReadAll reads r completely but fails when more than max bytes are available.
func ReadAll(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return data, err
	}
	if int64(len(data)) > max {
		return data[:max], fmt.Errorf("%w (mehr als %d Bytes)", ErrTooLarge, max)
	}
	return data, nil
}

// ReadFile reads a file but fails when it is larger than max bytes.
func ReadFile(path string, max int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ReadAll(file, max)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return data, nil
}

// Truncate shortens text to at most max bytes for log and error output,
// without splitting a UTF-8 sequence.
func Truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && cut < len(text) && text[cut]&0xC0 == 0x80 {
		cut--
	}
	return text[:cut] + " …(gekürzt)"
}
