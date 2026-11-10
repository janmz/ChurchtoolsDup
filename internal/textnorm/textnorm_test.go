package textnorm

import "testing"

func TestFoldAlnum(t *testing.T) {
	cases := map[string]string{
		"Süd":              "sued",
		"ÄÖÜ ß":            "aeoeuess",
		"Café Noël":        "cafenoel",
		"Île-de-France":    "iledefrance",
		"Standort 2":       "standort2",
		"  Frankfurt/M. 3": "frankfurtm3",
		"Ü":                "ue",
	}
	for in, want := range cases {
		if got := FoldAlnum(in); got != want {
			t.Errorf("FoldAlnum(%q) = %q, want %q", in, got, want)
		}
	}
}
