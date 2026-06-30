package helpers

import (
	"regexp"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

// richTextPolicy = kebijakan sanitasi HTML rich-text (output Trumbowyg) sebelum
// disimpan/ditampilkan — padanan helpers/sanitizeHtml.ts (cleanRichText) di
// NodeAdmin. Whitelist tag aman + atribut; buang <script>, event handler
// (onerror, dll), dan skema URL berbahaya (javascript:). Cegah XSS.
var (
	richTextPolicy *bluemonday.Policy
	richTextOnce   sync.Once
)

func buildRichTextPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "div", "span", "b", "i", "u", "strong", "em", "del", "s",
		"sup", "sub", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote",
		"ul", "ol", "li", "hr", "pre", "code",
		"table", "thead", "tbody", "tr", "th", "td",
	)
	// Link: href aman (http/https/mailto) + rel noopener.
	p.AllowAttrs("href", "name", "target").OnElements("a")
	p.AllowStandardURLs()
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	p.AllowURLSchemes("http", "https", "mailto")
	// Gambar: src (http/https/data utk inline) + atribut tampilan.
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")
	p.AllowImages() // izinkan data: URI khusus gambar
	// style terbatas (text-align, width/max-width) di semua elemen.
	p.AllowAttrs("style").Globally()
	p.AllowStyles("text-align").MatchingEnum("left", "right", "center", "justify").Globally()
	p.AllowStyles("width", "max-width").Matching(regexp.MustCompile(`^\d+(\.\d+)?(px|%)$`)).Globally()
	return p
}

// CleanRichText menyanitasi HTML rich-text (mis. deskripsi Setting dari editor)
// agar aman dirender mentah. String kosong → kosong.
func CleanRichText(dirty string) string {
	if dirty == "" {
		return ""
	}
	richTextOnce.Do(func() { richTextPolicy = buildRichTextPolicy() })
	return richTextPolicy.Sanitize(dirty)
}
