package tg

import (
	"testing"

	gtraw "github.com/gotd/td/tg"
)

func TestMessageSummaryFromClassIncludesAttachmentsAndButtons(t *testing.T) {
	message := &gtraw.Message{
		ID:      101,
		Message: "hola",
		Date:    1_712_345_678,
		Media: &gtraw.MessageMediaPhoto{
			Spoiler: true,
		},
		ReplyMarkup: &gtraw.ReplyInlineMarkup{
			Rows: []gtraw.KeyboardButtonRow{
				{
					Buttons: []gtraw.KeyboardButtonClass{
						&gtraw.KeyboardButtonCallback{Text: "Confirmar", Data: []byte("ok")},
						&gtraw.KeyboardButtonURL{Text: "Abrir", URL: "https://example.com"},
					},
				},
			},
		},
	}

	summary, ok := messageSummaryFromClass(message)
	if !ok {
		t.Fatalf("messageSummaryFromClass() ok = false, want true")
	}

	if got := len(summary.Attachments); got != 1 {
		t.Fatalf("attachments len = %d, want 1", got)
	}
	if got := summary.Attachments[0].Kind; got != "photo" {
		t.Fatalf("attachment kind = %q, want photo", got)
	}
	if got := len(summary.Buttons); got != 2 {
		t.Fatalf("buttons len = %d, want 2", got)
	}
	if got := summary.Buttons[0].Kind; got != "callback" {
		t.Fatalf("first button kind = %q, want callback", got)
	}
	if got := summary.Buttons[1].URL; got != "https://example.com" {
		t.Fatalf("second button url = %q, want https://example.com", got)
	}
}

func TestMessageSummaryFromClassClassifiesVoiceDocument(t *testing.T) {
	message := &gtraw.Message{
		ID:      102,
		Message: "",
		Date:    1_712_345_679,
		Media: &gtraw.MessageMediaDocument{
			Voice: true,
			Document: &gtraw.Document{
				ID:       501,
				MimeType: "audio/ogg",
				Size:     2048,
				Attributes: []gtraw.DocumentAttributeClass{
					&gtraw.DocumentAttributeFilename{FileName: "voice.ogg"},
				},
			},
		},
	}

	summary, ok := messageSummaryFromClass(message)
	if !ok {
		t.Fatalf("messageSummaryFromClass() ok = false, want true")
	}

	if got := len(summary.Attachments); got != 1 {
		t.Fatalf("attachments len = %d, want 1", got)
	}
	if got := summary.Attachments[0].Kind; got != "voice" {
		t.Fatalf("attachment kind = %q, want voice", got)
	}
	if got := summary.Attachments[0].Details["fileName"]; got != "voice.ogg" {
		t.Fatalf("attachment details.fileName = %v, want voice.ogg", got)
	}
}

func TestMessageSummaryFromClassClassifiesDocumentVariants(t *testing.T) {
	testCases := []struct {
		name    string
		media   *gtraw.MessageMediaDocument
		want    string
		detailK string
		detailV any
	}{
		{
			name: "plain document",
			media: &gtraw.MessageMediaDocument{
				Document: &gtraw.Document{
					ID:       601,
					MimeType: "application/pdf",
					Size:     4096,
					Attributes: []gtraw.DocumentAttributeClass{
						&gtraw.DocumentAttributeFilename{FileName: "manual.pdf"},
					},
				},
			},
			want:    "document",
			detailK: "fileName",
			detailV: "manual.pdf",
		},
		{
			name: "video document",
			media: &gtraw.MessageMediaDocument{
				Video: true,
				Document: &gtraw.Document{
					ID:       602,
					MimeType: "video/mp4",
					Size:     8192,
				},
			},
			want:    "video",
			detailK: "mimeType",
			detailV: "video/mp4",
		},
		{
			name: "audio document",
			media: &gtraw.MessageMediaDocument{
				Document: &gtraw.Document{
					ID:       603,
					MimeType: "audio/mpeg",
					Size:     2048,
				},
			},
			want:    "audio",
			detailK: "mimeType",
			detailV: "audio/mpeg",
		},
		{
			name: "sticker document",
			media: &gtraw.MessageMediaDocument{
				Document: &gtraw.Document{
					ID:       604,
					MimeType: "image/webp",
					Size:     1024,
					Attributes: []gtraw.DocumentAttributeClass{
						&gtraw.DocumentAttributeSticker{Alt: "🙂"},
					},
				},
			},
			want:    "sticker",
			detailK: "alt",
			detailV: "🙂",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := &gtraw.Message{
				ID:      200,
				Message: "",
				Date:    1_712_345_700,
				Media:   tc.media,
			}

			summary, ok := messageSummaryFromClass(message)
			if !ok {
				t.Fatalf("messageSummaryFromClass() ok = false, want true")
			}
			if got := len(summary.Attachments); got != 1 {
				t.Fatalf("attachments len = %d, want 1", got)
			}
			if got := summary.Attachments[0].Kind; got != tc.want {
				t.Fatalf("attachment kind = %q, want %q", got, tc.want)
			}
			if got := summary.Attachments[0].Details[tc.detailK]; got != tc.detailV {
				t.Fatalf("attachment details[%q] = %v, want %v", tc.detailK, got, tc.detailV)
			}
		})
	}
}

func TestSelectInlineButtonPrefersIndexAndFallsBackToText(t *testing.T) {
	buttons := []inlineButtonOption{
		{
			Summary: InlineButtonSummary{
				Index: 0,
				Text:  "Duplicado",
				Kind:  "callback",
			},
		},
		{
			Summary: InlineButtonSummary{
				Index: 1,
				Text:  "Duplicado",
				Kind:  "url",
				URL:   "https://example.com",
			},
		},
	}

	selected, err := selectInlineButton(buttons, PressButtonRequest{
		ButtonIndex:    1,
		HasButtonIndex: true,
		ButtonText:     "Duplicado",
	})
	if err != nil {
		t.Fatalf("selectInlineButton() error = %v, want nil", err)
	}
	if got := selected.Summary.Index; got != 1 {
		t.Fatalf("selectInlineButton() index = %d, want 1", got)
	}

	_, err = selectInlineButton(buttons, PressButtonRequest{ButtonText: "Duplicado"})
	if err == nil || err != ErrButtonAmbiguous {
		t.Fatalf("selectInlineButton() error = %v, want ErrButtonAmbiguous", err)
	}

	_, err = selectInlineButton(buttons, PressButtonRequest{ButtonText: "Inexistente"})
	if err == nil || err != ErrButtonNotFound {
		t.Fatalf("selectInlineButton() error = %v, want ErrButtonNotFound", err)
	}
}

func TestCallbackAnswerSummaryFromResponse(t *testing.T) {
	response := &gtraw.MessagesBotCallbackAnswer{
		Alert:     true,
		HasURL:    true,
		NativeUI:  false,
		Message:   "hecho",
		URL:       "https://example.com/next",
		CacheTime: 15,
	}

	summary := callbackAnswerSummaryFromResponse(response)
	if summary == nil {
		t.Fatalf("callbackAnswerSummaryFromResponse() = nil, want summary")
	}
	if got := summary.Message; got != "hecho" {
		t.Fatalf("summary.Message = %q, want hecho", got)
	}
	if got := summary.URL; got != "https://example.com/next" {
		t.Fatalf("summary.URL = %q, want https://example.com/next", got)
	}
}
