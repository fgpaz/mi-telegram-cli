package app

import (
	"fmt"
	"strings"
	"time"

	"mi-telegram-cli/internal/tg"
	qrlib "rsc.io/qr"
)

const qrQuietZone = 2

func renderQRLoginBlock(token tg.QRLoginToken, now func() time.Time) (string, int, error) {
	if now == nil {
		now = time.Now
	}

	rendered, err := renderCompactQR(token.URL)
	if err != nil {
		return "", 0, err
	}

	remaining := token.ExpiresAt.Sub(now())
	if remaining < 0 {
		remaining = 0
	}
	remaining = remaining.Truncate(time.Second)

	block := fmt.Sprintf(
		"[mi-telegram-cli] Telegram QR Login\n"+
			"Scan with Telegram > Settings > Devices > Link Desktop Device.\n"+
			"Expires in: %s\n"+
			"Fallback link:\n"+
			"%s\n"+
			"QR refreshes automatically in this terminal until accepted or timeout.\n\n"+
			"%s",
		remaining,
		token.URL,
		rendered,
	)

	return block, countRenderedLines(block), nil
}

func renderCompactQR(content string) (string, error) {
	code, err := qrlib.Encode(content, qrlib.M)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	maxX := code.Size + qrQuietZone
	maxY := code.Size + qrQuietZone
	for y := -qrQuietZone; y < maxY; y += 2 {
		for x := -qrQuietZone; x < maxX; x++ {
			top := qrModuleBlack(code, x, y)
			bottom := qrModuleBlack(code, x, y+1)
			switch {
			case top && bottom:
				builder.WriteRune('█')
			case top:
				builder.WriteRune('▀')
			case bottom:
				builder.WriteRune('▄')
			default:
				builder.WriteRune(' ')
			}
		}
		builder.WriteByte('\n')
	}

	return builder.String(), nil
}

func qrModuleBlack(code *qrlib.Code, x, y int) bool {
	if x < 0 || x >= code.Size || y < 0 || y >= code.Size {
		return false
	}
	return code.Black(x, y)
}

func countRenderedLines(value string) int {
	if value == "" {
		return 0
	}

	trimmed := strings.TrimSuffix(value, "\n")
	return strings.Count(trimmed, "\n") + 1
}
