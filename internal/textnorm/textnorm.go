// Package textnorm folds user-entered text into a comparable ASCII form.
package textnorm

import (
	"strings"
	"unicode"
)

// transliterations maps lower-case non-ASCII letters to ASCII replacements.
// German umlauts use their conventional two-letter spelling (ü → ue); other
// accented letters lose their diacritic (é → e, î → i).
var transliterations = map[rune]string{
	'ä': "ae", 'ö': "oe", 'ü': "ue", 'ß': "ss", 'æ': "ae", 'œ': "oe",
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c", 'ĉ': "c", 'ċ': "c",
	'ď': "d", 'đ': "d", 'ð': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ğ': "g", 'ģ': "g",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i", 'ı': "i",
	'ķ': "k",
	'ł': "l", 'ľ': "l", 'ĺ': "l", 'ļ': "l",
	'ñ': "n", 'ń': "n", 'ň': "n", 'ņ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ŕ': "r", 'ř': "r",
	'ś': "s", 'š': "s", 'ş': "s", 'ș': "s",
	'ť': "t", 'ţ': "t", 'ț': "t", 'þ': "th",
	'ù': "u", 'ú': "u", 'û': "u", 'ů': "u", 'ū': "u", 'ű': "u", 'ų': "u",
	'ý': "y", 'ÿ': "y",
	'ź': "z", 'ż': "z", 'ž': "z",
}

// FoldAlnum lower-cases value, transliterates umlauts and accented letters to
// ASCII and keeps only the letters a–z and digits 0–9. Spaces, hyphens and
// punctuation are dropped, so "Süd-Ost 2" becomes "suedost2".
func FoldAlnum(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r < unicode.MaxASCII:
			// other ASCII characters (spaces, punctuation) are dropped
		default:
			if repl, ok := transliterations[r]; ok {
				b.WriteString(repl)
			}
		}
	}
	return b.String()
}
