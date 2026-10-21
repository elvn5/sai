package screens

import (
	"html"
	"strconv"
	"strings"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// pageLimit is how much text a page of an article or a tour holds: a
// message takes 4096 characters, the rest is the page's heading.
const pageLimit = 3300

// inlineTags are the formatting Telegram's HTML keeps, by element.
var inlineTags = map[atom.Atom]string{
	atom.B: "b", atom.Strong: "b", atom.I: "i", atom.Em: "i", atom.U: "u",
	atom.S: "s", atom.Del: "s", atom.Strike: "s", atom.Code: "code",
}

// rich turns the site's HTML (the editor's output) into paragraphs of
// Telegram HTML, each with its tags closed.
type rich struct {
	blocks []string
	cur    strings.Builder
}

// Paragraphs converts HTML content into Telegram HTML paragraphs: headings
// turn bold, lists get bullets, pictures and anything Telegram can't show
// are dropped.
func Paragraphs(content string) []string {
	nodes, err := xhtml.ParseFragment(strings.NewReader(content), &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Body, Data: "body"})
	if err != nil {
		return nil
	}
	var r rich
	for _, n := range nodes {
		r.node(n, false)
	}
	r.flush()
	return r.blocks
}

// PlainParagraphs splits plain text into escaped paragraphs.
func PlainParagraphs(text string) []string {
	var out []string
	for _, p := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, esc(p))
		}
	}
	return out
}

func (r *rich) flush() {
	if s := strings.TrimSpace(r.cur.String()); s != "" && strings.TrimSpace(stripTags(s)) != "" {
		r.blocks = append(r.blocks, s)
	}
	r.cur.Reset()
}

// text writes text with whitespace collapsed, as a browser shows it.
func (r *rich) text(s string) { r.cur.WriteString(esc(strings.Join(strings.Fields(s), " "))) }

func (r *rich) children(n *xhtml.Node, inline bool) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.node(c, inline)
	}
}

// node renders n; inline is set inside a list item or a quote, where
// paragraphs don't break the block.
func (r *rich) node(n *xhtml.Node, inline bool) {
	switch n.Type {
	case xhtml.TextNode:
		r.spaced(n.Data)
		return
	case xhtml.ElementNode:
	default:
		return
	}
	switch n.DataAtom {
	case atom.Script, atom.Style, atom.Img, atom.Iframe, atom.Figure, atom.Video, atom.Audio, atom.Svg:
		if n.DataAtom == atom.Figure {
			// A figure's caption reads on without the picture.
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.DataAtom == atom.Figcaption {
					r.block(c, inline, "<i>", "</i>")
				}
			}
		}
	case atom.Br:
		r.cur.WriteString("\n")
	case atom.Hr:
		if !inline {
			r.flush()
		}
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		r.block(n, inline, "<b>", "</b>")
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Figcaption:
		r.block(n, inline, "", "")
	case atom.Blockquote:
		if inline {
			r.children(n, true)
			return
		}
		r.flush()
		r.cur.WriteString("<blockquote>")
		r.children(n, true)
		r.trimEnd()
		r.cur.WriteString("</blockquote>")
		r.flush()
	case atom.Pre:
		if !inline {
			r.flush()
		}
		r.cur.WriteString("<pre>" + esc(strings.Trim(textOf(n), "\n")) + "</pre>")
		if !inline {
			r.flush()
		}
	case atom.Ul, atom.Ol:
		r.list(n, inline)
	case atom.A:
		href := attr(n, "href")
		if strings.HasPrefix(href, "https://") || strings.HasPrefix(href, "http://") {
			r.cur.WriteString(`<a href="` + esc(href) + `">`)
			r.children(n, true)
			r.cur.WriteString("</a>")
			return
		}
		r.children(n, true)
	default:
		if tag, ok := inlineTags[n.DataAtom]; ok {
			r.cur.WriteString("<" + tag + ">")
			r.children(n, true)
			r.cur.WriteString("</" + tag + ">")
			return
		}
		r.children(n, inline)
	}
}

// block renders a paragraph-like element; inside a list item it only
// starts a new line.
func (r *rich) block(n *xhtml.Node, inline bool, open, close string) {
	if inline {
		if s := r.cur.String(); s != "" && !strings.HasSuffix(s, "\n") && !strings.HasSuffix(s, "• ") && !strings.HasSuffix(s, ". ") &&
			!strings.HasSuffix(s, "<blockquote>") {
			r.cur.WriteString("\n")
		}
		r.cur.WriteString(open)
		r.children(n, true)
		r.trimEnd()
		r.cur.WriteString(close)
		return
	}
	r.flush()
	r.cur.WriteString(open)
	r.children(n, true)
	r.trimEnd()
	r.cur.WriteString(close)
	r.flush()
}

// list renders a list as one paragraph, an item per line ("• ", "1. ");
// a nested list indents.
func (r *rich) list(n *xhtml.Node, inline bool) {
	indent := ""
	if inline {
		indent = "   "
		r.trimEnd()
	} else {
		r.flush()
	}
	i := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.DataAtom != atom.Li {
			continue
		}
		i++
		if r.cur.Len() > 0 {
			r.cur.WriteString("\n")
		}
		marker := "• "
		if n.DataAtom == atom.Ol {
			marker = strconv.Itoa(i) + ". "
		}
		r.cur.WriteString(indent + marker)
		r.children(c, true)
		r.trimEnd()
	}
	if !inline {
		r.flush()
	}
}

// spaced writes a text node, keeping one space where the HTML had
// whitespace between words.
func (r *rich) spaced(s string) {
	if strings.TrimSpace(s) == "" {
		if s != "" {
			r.space()
		}
		return
	}
	if startsSpace(s) {
		r.space()
	}
	r.text(s)
	if endsSpace(s) {
		r.cur.WriteString(" ")
	}
}

func (r *rich) space() {
	raw := r.cur.String()
	if raw != "" && !strings.HasSuffix(raw, " ") && !strings.HasSuffix(raw, "\n") {
		r.cur.WriteString(" ")
	}
}

// trimEnd drops trailing spaces before a closing tag or line end.
func (r *rich) trimEnd() {
	s := strings.TrimRight(r.cur.String(), " ")
	r.cur.Reset()
	r.cur.WriteString(s)
}

func startsSpace(s string) bool {
	c, _ := utf8.DecodeRuneInString(s)
	return strings.ContainsRune(" \t\n\r\f", c)
}

func endsSpace(s string) bool {
	c, _ := utf8.DecodeLastRuneInString(s)
	return strings.ContainsRune(" \t\n\r\f", c)
}

func attr(n *xhtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textOf(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// stripTags is the visible text of Telegram HTML.
func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return html.UnescapeString(b.String())
}

// Pages packs paragraphs into pages of up to limit visible characters; a
// paragraph too long for a page goes on as plain text, split between
// words, filling the page it starts on.
func Pages(paragraphs []string, limit int) []string {
	var pages, cur []string
	size := 0
	flush := func() {
		if len(cur) > 0 {
			pages = append(pages, strings.Join(cur, "\n\n"))
		}
		cur, size = nil, 0
	}
	add := func(p string, n int) {
		if len(cur) > 0 {
			size += 2
		}
		cur = append(cur, p)
		size += n
	}
	for _, p := range paragraphs {
		n := utf8.RuneCountInString(stripTags(p))
		if n <= limit {
			if len(cur) > 0 && size+2+n > limit {
				flush()
			}
			add(p, n)
			continue
		}
		room := limit
		if len(cur) > 0 {
			room = limit - size - 2
			if room < limit/4 {
				flush()
				room = limit
			}
		}
		for i, chunk := range splitWords(stripTags(p), room, limit) {
			if i > 0 {
				flush()
			}
			add(esc(chunk), utf8.RuneCountInString(chunk))
		}
	}
	flush()
	return pages
}

// splitWords cuts text into a chunk of up to first characters, then
// chunks of up to limit: after a sentence when one ends in the chunk's
// last part, else between words.
func splitWords(text string, first, limit int) []string {
	var out []string
	var words []string
	n, max := 0, first
	cut := func() {
		// The last sentence end past 60% of the chunk.
		at, size := -1, 0
		for i, w := range words[:len(words)-1] {
			size += utf8.RuneCountInString(w) + 1
			if size*5 >= max*3 && strings.ContainsAny(w[len(w)-1:], ".!?…") {
				at = i
			}
		}
		if at < 0 {
			at = len(words) - 1
		}
		out = append(out, strings.Join(words[:at+1], " "))
		words = append([]string(nil), words[at+1:]...)
		n = 0
		for i, w := range words {
			if i > 0 {
				n++
			}
			n += utf8.RuneCountInString(w)
		}
		max = limit
	}
	for _, word := range strings.Fields(text) {
		w := utf8.RuneCountInString(word)
		if len(words) > 0 && n+1+w > max {
			cut()
		}
		if len(words) > 0 {
			n++
		}
		words = append(words, word)
		n += w
	}
	if len(words) > 0 {
		out = append(out, strings.Join(words, " "))
	}
	return out
}
