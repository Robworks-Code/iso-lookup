package parse

import (
	"os"
	"strings"

	"github.com/ringo380/iso-lookup/internal/docmodel"
	"golang.org/x/net/html"
)

func parseHTML(path string) (Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	root, err := html.Parse(strings.NewReader(string(b)))
	if err != nil {
		return Document{}, err
	}
	var flat []Section
	cur := -1
	var title string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				if title == "" {
					title = strings.TrimSpace(textOf(n))
				}
				return
			case "h2", "h3", "h4", "h5", "h6":
				num, ttl := docmodel.SplitNumber(textOf(n))
				flat = append(flat, Section{Number: num, Title: ttl})
				cur = len(flat) - 1
				return
			case "p", "li":
				if cur >= 0 {
					t := strings.TrimSpace(textOf(n))
					if t != "" {
						if flat[cur].Body != "" {
							flat[cur].Body += "\n\n"
						}
						flat[cur].Body += t
					}
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(root)
	doc := Document{Title: title, Raw: textOf(root)}
	if len(flat) == 0 {
		doc.Sections = []Section{{Body: strings.TrimSpace(doc.Raw)}}
	} else {
		doc.Sections = docmodel.Nest(flat)
	}
	return doc, nil
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			sb.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}
