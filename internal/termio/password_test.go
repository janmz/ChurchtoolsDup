package termio

import (
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
		{name: "eof without newline", input: "only", wantErr: true},
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
