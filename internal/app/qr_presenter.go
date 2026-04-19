package app

import (
	"fmt"
	"io"
	"time"

	"mi-telegram-cli/internal/tg"
)

type qrLoginPresenter struct {
	w             io.Writer
	now           func() time.Time
	supportsANSI  bool
	renderedLines int
}

func newQRLoginPresenter(w io.Writer, now func() time.Time, supportsANSI bool) *qrLoginPresenter {
	return &qrLoginPresenter{
		w:            w,
		now:          now,
		supportsANSI: supportsANSI,
	}
}

func (p *qrLoginPresenter) Show(token tg.QRLoginToken) error {
	block, lineCount, err := renderQRLoginBlock(token, p.now)
	if err != nil {
		return err
	}

	if p.supportsANSI && p.renderedLines > 0 {
		if _, err := fmt.Fprintf(p.w, "\x1b[%dA\x1b[J", p.renderedLines); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(p.w, block); err != nil {
		return err
	}

	p.renderedLines = lineCount
	return nil
}
