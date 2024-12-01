package gemini

import (
	"strings"

	gemtext "git.sr.ht/~kota/goldmark-gemtext"
)

// WikilinkReplacer is a LinkReplacer that changes spaces to dashes and
// converts the link to lowercase. It is used to convert wiki links to
// gemini in the preferred canonical format (which is also used by the Builder
// for filenames when outputting files).
var WikilinkReplacer = gemtext.LinkReplacer{
	Function: func(input string) string {
		str := strings.ToLower(input)
		str = strings.ReplaceAll(str, " ", "-")
		return str + ".gmi"
	},
	Type: gemtext.LinkWiki,
}
