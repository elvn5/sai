// Package sanitize cleans rich text from the TipTap editor before it's
// stored. The frontends render it as HTML, so anything not explicitly
// allowed (scripts, event handlers, javascript: URLs, other tags) must be
// removed.
//
// It reproduces the output of the sanitize-html configuration the former
// NestJS API used (so stored content stays consistent), including its
// serialization: `<br />`, links forced to open in a new tab, entities
// decoded and only & < > (and " in attributes) re-escaped. testdata holds
// sanitize-html's own output for the comparison.
package sanitize

import (
	"errors"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	allowedTags = set("p", "br", "strong", "em", "s", "u", "code", "pre", "blockquote", "h2", "h3", "ul", "ol", "li", "a", "img")
	// Allowed attributes per tag, in no particular order.
	allowedAttrs = map[string]map[string]bool{
		"a":   set("href", "target", "rel"),
		"img": set("src", "alt", "title"),
	}
	urlAttrs       = set("href", "src")
	allowedSchemes = set("http", "https")
	// Content of these is dropped along with the tag.
	nonTextTags = set("script", "style", "textarea", "option", "noscript", "iframe")
	selfClosing = set("img", "br", "hr", "area", "base", "basefont", "input", "link", "meta")
	// Kept even when empty (sanitize-html's allowedEmptyAttributes).
	emptyAllowed = set("alt")

	schemePattern = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9.\-+]*):`)
	// Characters sanitize-html strips before looking for a URL scheme.
	urlNoise = regexp.MustCompile(`[\x00-\x20]+`)
)

type attr struct{ key, val string }

// HTML returns the sanitized form of raw.
func HTML(raw string) string {
	var out strings.Builder
	z := html.NewTokenizer(strings.NewReader(raw))
	var open []string // allowed tags currently open, for closing at EOF
	skipDepth := 0    // >0 while inside a non-text tag

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				for i := len(open) - 1; i >= 0; i-- {
					out.WriteString("</" + open[i] + ">")
				}
			}
			return out.String()

		case html.TextToken:
			if skipDepth == 0 {
				out.WriteString(escapeText(string(z.Text())))
			}

		case html.StartTagToken, html.SelfClosingTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if nonTextTags[name] {
				if tt == html.StartTagToken {
					skipDepth++
				}
				continue
			}
			if skipDepth > 0 || !allowedTags[name] {
				continue
			}
			writeStartTag(&out, name, tok.Attr)
			if selfClosing[name] {
				continue
			}
			open = append(open, name)

		case html.EndTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if nonTextTags[name] {
				if skipDepth > 0 {
					skipDepth--
				}
				continue
			}
			if skipDepth > 0 || !allowedTags[name] || selfClosing[name] {
				continue
			}
			// Close up to the matching open tag, like the parser does.
			for i := len(open) - 1; i >= 0; i-- {
				if open[i] != name {
					continue
				}
				for j := len(open) - 1; j >= i; j-- {
					out.WriteString("</" + open[j] + ">")
				}
				open = open[:i]
				break
			}
		}
		// Comments and doctypes are dropped.
	}
}

func writeStartTag(out *strings.Builder, name string, raw []html.Attribute) {
	attrs := make([]attr, 0, len(raw)+2)
	for _, a := range raw {
		attrs = append(attrs, attr{strings.ToLower(a.Key), a.Val})
	}
	if name == "a" {
		// transformTags: every link opens in a new tab, overriding the input.
		attrs = setAttr(attrs, "target", "_blank")
		attrs = setAttr(attrs, "rel", "noopener noreferrer")
	}

	out.WriteString("<" + name)
	for _, a := range attrs {
		if !allowedAttrs[name][a.key] {
			continue
		}
		if urlAttrs[a.key] && !safeURL(a.val) {
			continue
		}
		out.WriteString(" " + a.key)
		if a.val != "" {
			out.WriteString(`="` + escapeAttr(a.val) + `"`)
		} else if emptyAllowed[a.key] {
			out.WriteString(`=""`)
		}
	}
	if selfClosing[name] {
		out.WriteString(" />")
	} else {
		out.WriteString(">")
	}
}

// setAttr overwrites key in place, or appends it — matching how a JS
// object spread orders keys.
func setAttr(attrs []attr, key, val string) []attr {
	for i := range attrs {
		if attrs[i].key == key {
			attrs[i].val = val
			return attrs
		}
	}
	return append(attrs, attr{key, val})
}

// safeURL allows relative and protocol-relative URLs, and absolute URLs
// with an allowed scheme.
func safeURL(raw string) bool {
	cleaned := urlNoise.ReplaceAllString(raw, "")
	m := schemePattern.FindStringSubmatch(cleaned)
	if m == nil {
		return true
	}
	return allowedSchemes[strings.ToLower(m[1])]
}

var (
	textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
)

func escapeText(s string) string { return textEscaper.Replace(s) }
func escapeAttr(s string) string { return attrEscaper.Replace(s) }

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}
