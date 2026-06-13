package markdown

import (
	"fmt"
	"strings"
	"testing"
)

// TestRenderCache_HitReturnsSameResult guards I1: rendering the same input twice
// must return identical output and the cache must contain exactly one entry.
func TestRenderCache_HitReturnsSameResult(t *testing.T) {
	r := newTestRenderer()
	opts := Options{Width: 80, Styled: true}
	src := "Hello **world**"

	first := r.Render(src, opts)
	second := r.Render(src, opts)

	if first != second {
		t.Errorf("cache hit returned different output:\nfirst:  %q\nsecond: %q", first, second)
	}
	if got := len(r.cache); got != 1 {
		t.Errorf("expected 1 cache entry after two identical Render calls, got %d", got)
	}
}

// TestRenderCache_InlineKeyDistinguishedFromBlock guards I1 (the specific
// regression the reviewer named): Render and RenderInline for identical input
// must produce different cache entries. If the inline field were dropped from
// cacheKey, RenderInline could return a block-rendered (newline-containing)
// result.
//
// Two-paragraph input is used because it produces genuinely different output for
// block vs inline: block rendering joins paragraphs with "\n\n", while
// RenderInline flattens them to a space.
func TestRenderCache_InlineKeyDistinguishedFromBlock(t *testing.T) {
	r := newTestRenderer()
	opts := Options{Width: 80, Styled: false}
	src := "paragraph one\n\nparagraph two"

	blockOut := r.Render(src, opts)
	inlineOut := r.RenderInline(src, opts)

	if got := len(r.cache); got != 2 {
		t.Errorf("expected 2 cache entries after Render + RenderInline, got %d", got)
	}
	if blockOut == inlineOut {
		t.Errorf("Render and RenderInline returned identical output %q; inline key is not distinguishing", blockOut)
	}
	if strings.Contains(inlineOut, "\n") {
		t.Errorf("RenderInline returned a newline in %q; may have returned cached block output", inlineOut)
	}
}

// TestRenderCache_CappedAt256 guards I1: the cache must not grow unboundedly.
// After rendering 300 distinct strings the cache size must be <= 256, and
// rendering a fresh string after the flush must still return non-empty output.
func TestRenderCache_CappedAt256(t *testing.T) {
	r := newTestRenderer()
	opts := Options{Width: 80, Styled: false}

	for i := 0; i < 300; i++ {
		r.Render(fmt.Sprintf("unique string number %d", i), opts)
	}

	if got := len(r.cache); got > 256 {
		t.Errorf("cache grew past 256 entries: got %d", got)
	}

	// Rendering after a flush must still work.
	out := r.Render("post-flush render", opts)
	if out == "" {
		t.Error("Render returned empty string after cache flush")
	}
}
