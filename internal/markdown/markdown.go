package markdown

import (
	"image/color"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	text_pkg "github.com/yuin/goldmark/text"
)

// Theme holds styling colors for markdown rendering.
type Theme struct {
	Accent color.Color
	Dim    color.Color
	CodeBg color.Color
	Text   color.Color
}

// Options controls how markdown is rendered.
type Options struct {
	Width  int
	Styled bool
}

// cacheKey is used to deduplicate rendered output.
type cacheKey struct {
	text    string
	width   int
	styled  bool
	inline  bool
}

// Renderer renders markdown text to terminal-friendly strings.
type Renderer struct {
	md    goldmark.Markdown
	theme Theme
	cache map[cacheKey]string
}

// NewRenderer constructs a Renderer with GFM support and an empty cache.
func NewRenderer(theme Theme) *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	return &Renderer{
		md:    md,
		theme: theme,
		cache: make(map[cacheKey]string),
	}
}

// Render converts markdown text to a terminal string. Consults the internal
// cache before rendering; stores results on cache miss.
func (r *Renderer) Render(text string, opts Options) string {
	key := cacheKey{text: text, width: opts.Width, styled: opts.Styled, inline: false}
	if v, ok := r.cache[key]; ok {
		return v
	}
	if len(r.cache) >= 256 {
		r.cache = make(map[cacheKey]string)
	}
	result := r.doRender(text, opts)
	r.cache[key] = result
	return result
}

// RenderInline converts inline markdown to a terminal string. Consults the
// internal cache before rendering; stores results on cache miss.
func (r *Renderer) RenderInline(text string, opts Options) string {
	key := cacheKey{text: text, width: opts.Width, styled: opts.Styled, inline: true}
	if v, ok := r.cache[key]; ok {
		return v
	}
	if len(r.cache) >= 256 {
		r.cache = make(map[cacheKey]string)
	}
	result := r.doRenderInline(text, opts)
	r.cache[key] = result
	return result
}

// doRender parses and renders full block-level markdown.
func (r *Renderer) doRender(text string, opts Options) string {
	src := []byte(text)
	reader := text_pkg.NewReader(src)
	doc := r.md.Parser().Parse(reader)
	return renderBlocks(doc, src, r.theme, opts)
}

// doRenderInline is a stub for inline markdown rendering (AST walking added in T019).
func (r *Renderer) doRenderInline(text string, opts Options) string {
	return text
}
