package tg

import (
	"errors"
	"testing"

	gtraw "github.com/gotd/td/tg"
)

func TestPeerFromResolvedUsernameMapsBotUser(t *testing.T) {
	peer, err := peerFromResolvedUsername(&gtraw.ContactsResolvedPeer{
		Peer: &gtraw.PeerUser{UserID: 42},
		Users: []gtraw.UserClass{
			&gtraw.User{
				ID:         42,
				AccessHash: 99,
				Bot:        true,
				FirstName:  "Lab",
				Username:   "lab_bot",
			},
		},
	})
	if err != nil {
		t.Fatalf("peerFromResolvedUsername() error = %v, want nil", err)
	}
	if peer.Kind != "bot" || peer.ID != 42 || peer.Username != "lab_bot" || peer.DisplayName != "Lab" {
		t.Fatalf("peerFromResolvedUsername() = %+v, want bot user summary", peer)
	}
	inputPeer, ok := peer.Resolved.(*gtraw.InputPeerUser)
	if !ok {
		t.Fatalf("peer.Resolved = %T, want *InputPeerUser", peer.Resolved)
	}
	if inputPeer.UserID != 42 || inputPeer.AccessHash != 99 {
		t.Fatalf("input peer = %+v, want user_id/access_hash from resolved user", inputPeer)
	}
}

func TestPeerFromResolvedUsernameMapsChannel(t *testing.T) {
	peer, err := peerFromResolvedUsername(&gtraw.ContactsResolvedPeer{
		Peer: &gtraw.PeerChannel{ChannelID: 77},
		Chats: []gtraw.ChatClass{
			&gtraw.Channel{
				ID:         77,
				AccessHash: 88,
				Broadcast:  true,
				Title:      "Updates",
				Username:   "updates",
			},
		},
	})
	if err != nil {
		t.Fatalf("peerFromResolvedUsername() error = %v, want nil", err)
	}
	if peer.Kind != "channel" || peer.ID != 77 || peer.Username != "updates" || peer.DisplayName != "Updates" {
		t.Fatalf("peerFromResolvedUsername() = %+v, want channel summary", peer)
	}
	inputPeer, ok := peer.Resolved.(*gtraw.InputPeerChannel)
	if !ok {
		t.Fatalf("peer.Resolved = %T, want *InputPeerChannel", peer.Resolved)
	}
	if inputPeer.ChannelID != 77 || inputPeer.AccessHash != 88 {
		t.Fatalf("input peer = %+v, want channel_id/access_hash from resolved channel", inputPeer)
	}
}

func TestPeerFromResolvedUsernameMissingEntity(t *testing.T) {
	_, err := peerFromResolvedUsername(&gtraw.ContactsResolvedPeer{
		Peer: &gtraw.PeerUser{UserID: 42},
	})
	if !errors.Is(err, ErrPeerNotFound) {
		t.Fatalf("peerFromResolvedUsername() error = %v, want ErrPeerNotFound", err)
	}
}

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
