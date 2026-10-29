package glamour

import (
	"fmt"
	"strings"
	"testing"
)

func proseDoc(paragraphs int) string {
	var b strings.Builder
	for i := range paragraphs {
		fmt.Fprintf(&b, "Considering angle %d of the question in some detail, the constraint holds for every branch here and the wrapping needs to run across several lines to be representative.\n\n", i)
	}
	return b.String()
}

func mixedDoc(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "## Heading %d\n\nSome **bold** and _italic_ prose with `code` and a [link](https://example.com) that wraps.\n\n- item one\n- item two\n\n```go\nfunc f() { return }\n```\n\n", i)
	}
	return b.String()
}

func benchRender(b *testing.B, doc string) {
	r, err := NewTermRenderer(WithStandardStyle("dark"), WithWordWrap(120))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(doc)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := r.Render(doc); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderProse(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("paras=%d", n), func(b *testing.B) { benchRender(b, proseDoc(n)) })
	}
}

func BenchmarkRenderMixed(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("blocks=%d", n), func(b *testing.B) { benchRender(b, mixedDoc(n)) })
	}
}

func codeBlockDoc(language string, size int) string {
	const line = "value := compute(alpha, beta)\n"

	return "```" + language + "\n" + strings.Repeat(line, size/len(line)+1)[:size] + "```\n"
}

// BenchmarkRenderCodeBlock separates the two code-block paths: a fence with no
// info string selects no lexer, and one with an info string runs chroma over the
// whole body. The first must not pay for the lexer registry's analysers.
func BenchmarkRenderCodeBlock(b *testing.B) {
	for _, size := range []int{8 << 10, 32 << 10} {
		for _, language := range []string{"go", ""} {
			name := "labelled"
			if language == "" {
				name = "unlabelled"
			}
			b.Run(fmt.Sprintf("%s/bytes=%d", name, size), func(b *testing.B) {
				benchRender(b, codeBlockDoc(language, size))
			})
		}
	}
}
