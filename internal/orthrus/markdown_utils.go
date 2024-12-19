package orthrus

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/ast"
)

// GetMarkdownTitle returns the first heading in the document if found, or the empty
// string otherwise.
func GetMarkdownTitle(doc ast.Node, markdown []byte) (string, error) {
	var firstHeading string
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if heading, ok := n.(*ast.Heading); ok && entering {
			if heading.Level != 1 {
				return ast.WalkSkipChildren, nil
			}

			var buf bytes.Buffer
			for chld := heading.FirstChild(); chld != nil; chld = chld.NextSibling() {
				if text, ok := chld.(*ast.Text); ok {
					buf.Write(text.Segment.Value(markdown))
				}
			}
			firstHeading = buf.String()
			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})

	if firstHeading == "" {
		return "", fmt.Errorf("no heading found")
	}

	return firstHeading, nil
}
