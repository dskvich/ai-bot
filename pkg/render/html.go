package render

import (
	"bytes"
	"html"

	"github.com/russross/blackfriday"
)

type htmlRenderer struct {
	blackfriday.Renderer
}

func newHTMLRenderer() *htmlRenderer {
	return &htmlRenderer{
		blackfriday.HtmlRenderer(blackfriday.HTML_USE_XHTML|blackfriday.HTML_SKIP_IMAGES, "", ""),
	}
}

func (hr *htmlRenderer) Paragraph(out *bytes.Buffer, text func() bool) {
	marker := out.Len()
	if !text() {
		out.Truncate(marker)
		return
	}
	out.WriteString("\n\n")
}

func (hr *htmlRenderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	out.WriteString("<pre>")
	out.WriteString(html.EscapeString(string(text)))
	out.WriteString("</pre>\n")
}

func (hr *htmlRenderer) CodeSpan(out *bytes.Buffer, text []byte) {
	out.WriteString("<code>")
	out.WriteString(html.EscapeString(string(text)))
	out.WriteString("</code>")
}

func (hr *htmlRenderer) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	hr.Paragraph(out, text)
}

func (hr *htmlRenderer) HRule(out *bytes.Buffer) {
	out.WriteByte('\n')
}

func (hr *htmlRenderer) BlockQuote(out *bytes.Buffer, text []byte) {
	out.WriteString("> ")
	out.Write(text)
	out.WriteByte('\n')
}

func (hr *htmlRenderer) LineBreak(out *bytes.Buffer) {
	out.WriteByte('\n')
}

func (hr *htmlRenderer) List(out *bytes.Buffer, text func() bool, flags int) {
	hr.Paragraph(out, text)
}

func (hr *htmlRenderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	out.WriteString("- ")
	out.Write(text)
	out.WriteByte('\n')
}

func ToHTML(content string) string {
	renderer := newHTMLRenderer()
	extensions := blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS

	htmlOutput := blackfriday.Markdown([]byte(content), renderer, extensions)
	return string(htmlOutput)
}
