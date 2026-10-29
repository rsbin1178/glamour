package ansi

import (
	"io"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// penWriter forwards bytes to a destination while tracking the terminal pen, so
// a newline can close the style and link of the line it ends and reopen them on
// the next one.
//
// It is the contract of lipgloss's WrapWriter, which glamour used here: the same
// bytes in the same order. The difference is how they get there. WrapWriter
// writes one byte per call, so a rendered block costs one write and one
// allocation per byte of Markdown; this writer hands the printable run between
// two control bytes to the destination in a single call and only steps the
// parser byte by byte, which allocates nothing.
type penWriter struct {
	w     io.Writer
	p     *ansi.Parser
	style uv.Style
	link  uv.Link
}

// newPenWriter returns a pen writer over w.
func newPenWriter(w io.Writer) *penWriter {
	pw := &penWriter{w: w}

	pw.p = ansi.GetParser()
	pw.p.SetHandler(ansi.Handler{
		HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
			if cmd == 'm' {
				uv.ReadStyle(params, &pw.style)
			}
		},
		HandleOsc: func(cmd int, data []byte) {
			if cmd == 8 {
				uv.ReadLink(data, &pw.link)
			}
		},
	})

	return pw
}

// Style returns the current pen style.
func (w *penWriter) Style() uv.Style {
	return w.style
}

// Link returns the current pen link.
func (w *penWriter) Link() uv.Link {
	return w.link
}

// Write writes p to the destination, closing and reopening the pen around every
// newline it contains.
func (w *penWriter) Write(p []byte) (int, error) {
	if w.p == nil {
		// The writer was closed and its parser returned to the pool.
		return len(p), nil
	}

	start := 0

	for index := 0; index < len(p); index++ {
		if p[index] != '\n' {
			continue
		}

		// The parser has seen every byte before the newline, so the pen it holds
		// is the one that line ended with.
		w.advance(p[start:index])

		if err := w.emit(p[start:index]); err != nil {
			return 0, err
		}

		if err := w.closePen(); err != nil {
			return 0, err
		}

		if err := w.emit(p[index : index+1]); err != nil {
			return 0, err
		}

		if err := w.openPen(); err != nil {
			return 0, err
		}

		w.p.Advance(p[index])

		start = index + 1
	}

	if start < len(p) {
		w.advance(p[start:])

		if err := w.emit(p[start:]); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Close closes the pen and releases the parser.
func (w *penWriter) Close() error {
	if w.p == nil {
		return nil
	}

	if err := w.closePen(); err != nil {
		return err
	}

	ansi.PutParser(w.p)
	w.p = nil

	return nil
}

// advance steps the parser over one run of bytes, exactly once per byte.
func (w *penWriter) advance(p []byte) {
	for _, b := range p {
		w.p.Advance(b)
	}
}

// emit writes one run of bytes to the destination.
func (w *penWriter) emit(p []byte) error {
	if len(p) == 0 {
		return nil
	}

	if _, err := w.w.Write(p); err != nil {
		return err //nolint:wrapcheck // the caller wraps the writer chain once
	}

	return nil
}

// closePen ends the current line's style and link.
func (w *penWriter) closePen() error {
	if !w.style.IsZero() {
		if _, err := io.WriteString(w.w, ansi.ResetStyle); err != nil {
			return err //nolint:wrapcheck // the caller wraps the writer chain once
		}
	}

	if !w.link.IsZero() {
		if _, err := io.WriteString(w.w, ansi.ResetHyperlink()); err != nil {
			return err //nolint:wrapcheck // the caller wraps the writer chain once
		}
	}

	return nil
}

// openPen reopens the pen on the next line.
func (w *penWriter) openPen() error {
	if !w.link.IsZero() {
		if _, err := io.WriteString(w.w, ansi.SetHyperlink(w.link.URL, w.link.Params)); err != nil {
			return err //nolint:wrapcheck // the caller wraps the writer chain once
		}
	}

	if !w.style.IsZero() {
		if _, err := io.WriteString(w.w, w.style.String()); err != nil {
			return err //nolint:wrapcheck // the caller wraps the writer chain once
		}
	}

	return nil
}
