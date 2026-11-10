package termio

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestReadPasswordPlain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "unix newline", input: "secret\n", want: "secret"},
		{name: "windows newline", input: "secret\r\n", want: "secret"},
		{name: "empty line", input: "\n", want: ""},
		{name: "eof without newline", input: "geheim", want: "geheim"},
		{name: "keeps spaces", input: " pass word \n", want: " pass word "},
		{name: "keeps umlauts", input: "Müller2024\n", want: "Müller2024"},
		{name: "empty input", input: "", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := readPasswordPlain(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("readPasswordPlain: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// A shared buffered reader must deliver the password line even when an
// earlier prompt already buffered it (piped setup input).
func TestReadPasswordPlainSharedReader(t *testing.T) {
	in := bufio.NewReader(strings.NewReader("user\nsecret\n"))
	user, _ := in.ReadString('\n')
	if user != "user\n" {
		t.Fatalf("user = %q", user)
	}
	got, err := readPasswordPlain(in)
	if err != nil || got != "secret" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReadPasswordRawUTF8(t *testing.T) {
	// "Mü", backspace (removes the two-byte ü), "üß", Enter → "Müß".
	input := "Mü\x7füß\r"
	got, err := readPasswordRaw(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if got != "Müß" {
		t.Fatalf("got %q, want %q", got, "Müß")
	}
}

func TestReadLineLimit(t *testing.T) {
	long := strings.Repeat("x", MaxLineBytes+1) + "\n"
	if _, err := ReadLine(bufio.NewReader(strings.NewReader(long)), MaxLineBytes); err != ErrLineTooLong {
		t.Fatalf("err = %v, want ErrLineTooLong", err)
	}
	if _, err := readPasswordRaw(strings.NewReader(strings.Repeat("x", MaxLineBytes+1))); err != ErrLineTooLong {
		t.Fatalf("raw err = %v, want ErrLineTooLong", err)
	}
	got, err := ReadLine(bufio.NewReader(bytes.NewReader([]byte("ok\r\nnext"))), MaxLineBytes)
	if err != nil || got != "ok" {
		t.Fatalf("ReadLine = %q, %v", got, err)
	}
}
