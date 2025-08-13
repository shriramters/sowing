package renderer

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/niklasfasching/go-org/org"
)

func NewHTMLWriterWithChroma() *org.HTMLWriter {
	w := org.NewHTMLWriter()
	w.HighlightCodeBlock = func(source, lang string, inline bool, params map[string]string) string {
		var w bytes.Buffer
		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		iterator, err := lexer.Tokenise(nil, source)
		if err != nil {
			return source
		}
		formatter := html.New(html.WithClasses(true))
		if err := formatter.Format(&w, styles.Get("friendly"), iterator); err != nil {
			return source
		}
		return w.String()
	}
	return w
}
