package iolimit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadAllWithinLimit(t *testing.T) {
	data, err := ReadAll(strings.NewReader("abc"), 3)
	if err != nil || string(data) != "abc" {
		t.Fatalf("ReadAll = %q, %v", data, err)
	}
}

func TestReadAllTooLarge(t *testing.T) {
	_, err := ReadAll(strings.NewReader("abcd"), 3)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestReadFileTooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(path, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path, 5); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	if data, err := ReadFile(path, 10); err != nil || len(data) != 10 {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
}

func TestTruncateKeepsUTF8(t *testing.T) {
	got := Truncate("ääää", 3) // each ä is 2 bytes; cut must not split one
	if !strings.HasPrefix(got, "ä ") {
		t.Fatalf("Truncate = %q", got)
	}
	if Truncate("kurz", 10) != "kurz" {
		t.Fatal("short text must stay unchanged")
	}
}
