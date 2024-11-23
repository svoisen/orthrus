package gemini

import (
	"strings"

	gemtext "git.sr.ht/~kota/goldmark-gemtext"
)

var WikilinkReplacer = gemtext.LinkReplacer{
	Function: func(input string) string {
		str := strings.ToLower(input)
		str = strings.ReplaceAll(str, " ", "-")
		return str + ".gmi"
	},
	Type: gemtext.LinkWiki,
}
