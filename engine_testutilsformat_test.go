package wax_test

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func formatHTML(v string) string {
	buf := bytes.NewBufferString("")
	r := bytes.NewReader(([]byte)(v))
	node, err := html.Parse(r)
	if err != nil {
		panic(err)
	}

	prettyPrint(buf, node, 0)
	return buf.String()
}

func prettyPrint(b *bytes.Buffer, n *html.Node, depth int) {
	switch n.Type {
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			prettyPrint(b, c, depth)
		}

	case html.ElementNode:
		justRender := false
		switch {
		case n.FirstChild == nil:
			justRender = true
		case n.Data == "pre" || n.Data == "textarea":
			justRender = true
		case n.Data == "script" || n.Data == "style":
			break
		case n.FirstChild == n.LastChild && n.FirstChild.Type == html.TextNode:
			if !isInline(n) {
				c := n.FirstChild
				c.Data = strings.Trim(c.Data, " \t\n\r")
			}
			justRender = true
		case isInline(n) && contentIsInline(n):
			justRender = true
		}
		if justRender {
			indent(b, depth)
			html.Render(b, n)
			b.WriteByte('\n')
			return
		}
		indent(b, depth)
		fmt.Fprintln(b, html.Token{
			Type: html.StartTagToken,
			Data: n.Data,
			Attr: n.Attr,
		})
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if n.Data == "script" || n.Data == "style" && c.Type == html.TextNode {
				prettyPrintScript(b, c.Data, depth+1)
			} else {
				prettyPrint(b, c, depth+1)
			}
		}
		indent(b, depth)
		fmt.Fprintln(b, html.Token{
			Type: html.EndTagToken,
			Data: n.Data,
		})

	case html.TextNode:
		n.Data = strings.Trim(n.Data, " \t\n\r")
		if n.Data == "" {
			return
		}
		indent(b, depth)
		html.Render(b, n)
		b.WriteByte('\n')

	default:
		indent(b, depth)
		html.Render(b, n)
		b.WriteByte('\n')
	}
}

func isInline(n *html.Node) bool {
	switch n.Type {
	case html.TextNode, html.CommentNode:
		return true
	case html.ElementNode:
		switch n.Data {
		case "b", "big", "i", "small", "tt", "abbr", "acronym", "cite", "dfn", "em", "kbd", "strong", "samp", "var", "a", "bdo", "img", "map", "object", "q", "span", "sub", "sup", "button", "input", "label", "select", "textarea":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func contentIsInline(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !isInline(c) || !contentIsInline(c) {
			return false
		}
	}
	return true
}

func indent(b *bytes.Buffer, depth int) {
	depth *= 2
	for i := 0; i < depth; i++ {
		b.WriteByte(' ')
	}
}

func prettyPrintScript(b *bytes.Buffer, s string, depth int) {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		depthChange := 0
		for _, c := range line {
			switch c {
			case '(', '[', '{':
				depthChange++
			case ')', ']', '}':
				depthChange--
			}
		}
		switch line[0] {
		case '.':
			indent(b, depth+1)
		case ')', ']', '}':
			indent(b, depth-1)
		default:
			indent(b, depth)
		}
		depth += depthChange
		fmt.Fprintln(b, line)
	}
}
