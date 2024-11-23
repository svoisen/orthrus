package web

import (
	"path/filepath"
	"strings"

	"go.abhg.dev/goldmark/wikilink"
)

var _html = []byte(".html")
var _hash = []byte("#")

type WikilinkResolver struct{}

func transformTarget(input []byte) []byte {
	str := string(input)
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, " ", "-")
	return []byte(str)
}

func (WikilinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	newTarget := transformTarget(n.Target)
	dest := make([]byte, len(newTarget)+len(_html)+len(_hash)+len(n.Fragment))
	var i int
	if len(n.Target) > 0 {
		i += copy(dest, newTarget)
		if filepath.Ext(string(n.Target)) == "" {
			i += copy(dest[i:], _html)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	return dest[:i], nil
}
