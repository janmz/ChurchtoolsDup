package churchtools

import "testing"

func TestMainInstanceURL(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{
			name:   "sub instance",
			input:  "https://gemeinde-unterstadt.church.tools",
			want:   "https://gemeinde.church.tools",
			wantOK: true,
		},
		{
			name:   "sub instance trailing slash",
			input:  "https://gemeinde-unterstadt.church.tools/",
			want:   "https://gemeinde.church.tools",
			wantOK: true,
		},
		{
			name:   "main instance",
			input:  "https://gemeinde.church.tools",
			wantOK: false,
		},
		{
			name:   "custom domain",
			input:  "https://church.example.org",
			wantOK: false,
		},
		{
			name:   "no hyphen",
			input:  "https://gemeindeunterstadt.church.tools",
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MainInstanceURL(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if tc.wantOK && got != tc.want {
				t.Fatalf("url = %q, want %q", got, tc.want)
			}
		})
	}
}
