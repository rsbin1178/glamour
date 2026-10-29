package ansi

import (
	"bytes"
	"testing"

	"charm.land/lipgloss/v2"
)

// penWriterCorpus holds the byte shapes a rendered block puts through the writer
// chain: plain prose, styles, links, wide runes and the escapes glamour emits
// around padding.
func penWriterCorpus() map[string]string {
	return map[string]string{
		"plain":        "plain line without escapes\nand a second line\n",
		"no newline":   "one unterminated line",
		"styled":       "\x1b[1;31mred bold line\x1b[0m\nnext line\n",
		"style across": "\x1b[38;5;252mfirst line\nsecond line\nthird\x1b[m",
		"link":         "\x1b]8;;https://example.com\x1b\\linked\x1b]8;;\x1b\\\nafter\n",
		"link across":  "\x1b]8;;https://example.com\x07linked\nstill linked\x1b]8;;\x07",
		"wide runes":   "第一段包含中文与标点\n第二段继续说明\n",
		"emoji":        "first 🎯 line\nsecond 🚀 line\n",
		"blank lines":  "a\n\n\nb\n",
		"csi params":   "\x1b[38;5;252m\x1b[48;5;236mpadded cell\x1b[m\n",
		"split escape": "prefix \x1b[1m bold \x1b[0m suffix\nnext\n",
		"trailing esc": "text \x1b[",
	}
}

// TestPenWriterMatchesLipglossBytes pins the contract: the pen writer must emit
// exactly the bytes lipgloss's WrapWriter emits for the same writes, whatever
// the chunking, because glamour's padding and indentation depend on that output
// and every renderer golden was recorded through it.
func TestPenWriterMatchesLipglossBytes(t *testing.T) {
	t.Parallel()

	for name, document := range penWriterCorpus() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, chunk := range []int{1, 2, 3, 7, 64, len(document)} {
				if chunk <= 0 {
					continue
				}

				var want, got bytes.Buffer

				wantWriter := lipgloss.NewWrapWriter(&want)
				gotWriter := newPenWriter(&got)

				for start := 0; start < len(document); start += chunk {
					end := min(start+chunk, len(document))
					part := []byte(document[start:end])

					if _, err := wantWriter.Write(part); err != nil {
						t.Fatal(err)
					}

					if _, err := gotWriter.Write(part); err != nil {
						t.Fatal(err)
					}

					if wantWriter.Style() != gotWriter.Style() {
						t.Fatalf("chunk %d at %d: style %v != %v", chunk, start, wantWriter.Style(), gotWriter.Style())
					}

					if wantWriter.Link() != gotWriter.Link() {
						t.Fatalf("chunk %d at %d: link %v != %v", chunk, start, wantWriter.Link(), gotWriter.Link())
					}
				}

				if err := wantWriter.Close(); err != nil {
					t.Fatal(err)
				}

				if err := gotWriter.Close(); err != nil {
					t.Fatal(err)
				}

				if want.String() != got.String() {
					t.Fatalf("chunk %d:\nwant %q\n got %q", chunk, want.String(), got.String())
				}
			}
		})
	}
}

// TestPenWriterWritesRunsNotBytes pins the reason the writer exists: one write
// per line, not one per byte.
func TestPenWriterWritesRunsNotBytes(t *testing.T) {
	t.Parallel()

	counter := &writeCounter{}

	writer := newPenWriter(counter)
	_, err := writer.Write([]byte("a plain line of prose that is much longer than one byte\n"))
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if counter.calls > 3 {
		t.Fatalf("the line took %d writes; it is handed over in runs, not byte by byte", counter.calls)
	}
}

// writeCounter counts calls instead of keeping the bytes.
type writeCounter struct{ calls int }

func (c *writeCounter) Write(p []byte) (int, error) {
	c.calls++

	return len(p), nil
}
