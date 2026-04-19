package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/tg"
)

func (e *Executor) handleMessages(ctx context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing messages subcommand"), false
	}

	switch args[0] {
	case "read":
		fs := newFlagSet("messages read")
		profileID := fs.String("profile", "", "")
		peerQuery := fs.String("peer", "", "")
		limit := fs.Int("limit", 20, "")
		afterID := fs.Int64("after-id", 0, "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" || strings.TrimSpace(*peerQuery) == "" || *limit < 1 || *limit > 100 || *afterID < 0 {
			return e.errorResponse(*profileID, "InvalidInput", "invalid profile, peer, limit or after-id"), *jsonMode
		}
		return e.executeRead(ctx, *profileID, *peerQuery, *limit, *afterID, *jsonMode)
	case "send":
		fs := newFlagSet("messages send")
		profileID := fs.String("profile", "", "")
		peerQuery := fs.String("peer", "", "")
		text := fs.String("text", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		trimmedPeerQuery := strings.TrimSpace(*peerQuery)
		trimmedText := strings.TrimSpace(*text)
		if *profileID == "" || trimmedPeerQuery == "" || trimmedText == "" {
			return e.errorResponse(*profileID, "InvalidInput", "profile, peer and text are required"), *jsonMode
		}
		e.maybeWarnMSYSPathTranslation(trimmedText, *jsonMode)
		return e.executeSend(ctx, *profileID, trimmedPeerQuery, trimmedText, *jsonMode)
	case "wait":
		fs := newFlagSet("messages wait")
		profileID := fs.String("profile", "", "")
		peerQuery := fs.String("peer", "", "")
		afterID := fs.Int64("after-id", 0, "")
		timeoutSeconds := fs.Int("timeout", 0, "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" || strings.TrimSpace(*peerQuery) == "" || *timeoutSeconds < 1 || *timeoutSeconds > 300 || *afterID < 0 {
			return e.errorResponse(*profileID, "InvalidInput", "invalid profile, peer, after-id or timeout"), *jsonMode
		}
		return e.executeWait(ctx, *profileID, *peerQuery, *afterID, time.Duration(*timeoutSeconds)*time.Second, *jsonMode)
	case "press-button":
		fs := newFlagSet("messages press-button")
		profileID := fs.String("profile", "", "")
		peerQuery := fs.String("peer", "", "")
		messageID := fs.Int64("message-id", 0, "")
		buttonIndex := fs.Int("button-index", 0, "")
		buttonText := fs.String("button-text", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}

		hasButtonIndex := flagProvided(fs, "button-index")
		trimmedButtonText := strings.TrimSpace(*buttonText)
		if *profileID == "" || strings.TrimSpace(*peerQuery) == "" || *messageID < 1 || (!hasButtonIndex && trimmedButtonText == "") || (hasButtonIndex && *buttonIndex < 0) {
			return e.errorResponse(*profileID, "InvalidInput", "invalid profile, peer, message-id or button selector"), *jsonMode
		}

		return e.executePressButton(ctx, *profileID, *peerQuery, *messageID, *buttonIndex, hasButtonIndex, trimmedButtonText, *jsonMode)
	default:
		return e.errorResponse("", "InvalidInput", "unknown messages subcommand"), false
	}
}

func (e *Executor) executeRead(ctx context.Context, profileID, peerQuery string, limit int, afterID int64, jsonMode bool) (output.Response, bool) {
	return e.withProfileLock(profileID, jsonMode, func() output.Response {
		runtimeConfig, err := e.requireTelegramConfig()
		if err != nil {
			return e.errorResponse(profileID, "InvalidInput", err.Error())
		}

		sessionRef, err := e.authorizedSession(profileID)
		if err != nil {
			if errors.Is(err, errUnauthorizedProfile) {
				return e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized")
			}
			return e.mapStoreError(profileID, err)
		}

		peer, resp, ok := e.resolvePeer(ctx, profileID, runtimeConfig, sessionRef, peerQuery)
		if !ok {
			return resp
		}

		items, err := e.telegram.ReadMessages(ctx, runtimeConfig, sessionRef, tg.ReadMessagesRequest{
			Peer:    peer,
			Limit:   limit,
			AfterID: afterID,
		})
		if err != nil {
			return e.mapTelegramUnauthorizedOr(profileID, "TelegramReadFailed", err)
		}

		return output.Response{
			OK:      true,
			Profile: profileID,
			Data: map[string]any{
				"items": items,
				"count": len(items),
				"peer":  peer,
			},
		}
	})
}

func (e *Executor) executeSend(ctx context.Context, profileID, peerQuery, text string, jsonMode bool) (output.Response, bool) {
	return e.withProfileLock(profileID, jsonMode, func() output.Response {
		runtimeConfig, err := e.requireTelegramConfig()
		if err != nil {
			return e.errorResponse(profileID, "InvalidInput", err.Error())
		}

		sessionRef, err := e.authorizedSession(profileID)
		if err != nil {
			if errors.Is(err, errUnauthorizedProfile) {
				return e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized")
			}
			return e.mapStoreError(profileID, err)
		}

		peer, resp, ok := e.resolvePeer(ctx, profileID, runtimeConfig, sessionRef, peerQuery)
		if !ok {
			return resp
		}

		result, err := e.telegram.SendMessage(ctx, runtimeConfig, sessionRef, tg.SendMessageRequest{
			Peer: peer,
			Text: text,
		})
		if err != nil {
			return e.mapTelegramUnauthorizedOr(profileID, "TelegramSendFailed", err)
		}

		return output.Response{
			OK:      true,
			Profile: profileID,
			Data: map[string]any{
				"peer":      peer,
				"messageId": result.MessageID,
				"sentAtUtc": result.SentAtUTC,
			},
		}
	})
}

func (e *Executor) maybeWarnMSYSPathTranslation(text string, jsonMode bool) {
	if jsonMode {
		return
	}
	if !looksLikeMSYSPathTranslatedText(text) {
		return
	}

	_, _ = fmt.Fprintln(e.stderr, "warning: possible MSYS path translation detected - prepend MSYS_NO_PATHCONV=1")
}

func looksLikeMSYSPathTranslatedText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	suspiciousPrefixes := []string{
		"c:/program files/git/",
		"c:/program files (x86)/git/",
		"c:/msys64/",
		"/mingw64/",
		"/mingw32/",
		"/ucrt64/",
		"/clang64/",
	}

	for _, prefix := range suspiciousPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}

	return false
}

func (e *Executor) executeWait(ctx context.Context, profileID, peerQuery string, afterID int64, timeout time.Duration, jsonMode bool) (output.Response, bool) {
	return e.withProfileLock(profileID, jsonMode, func() output.Response {
		runtimeConfig, err := e.requireTelegramConfig()
		if err != nil {
			return e.errorResponse(profileID, "InvalidInput", err.Error())
		}

		sessionRef, err := e.authorizedSession(profileID)
		if err != nil {
			if errors.Is(err, errUnauthorizedProfile) {
				return e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized")
			}
			return e.mapStoreError(profileID, err)
		}

		peer, resp, ok := e.resolvePeer(ctx, profileID, runtimeConfig, sessionRef, peerQuery)
		if !ok {
			return resp
		}

		message, err := e.telegram.WaitMessage(ctx, runtimeConfig, sessionRef, tg.WaitMessageRequest{
			Peer:    peer,
			AfterID: afterID,
			Timeout: timeout,
		})
		if err != nil {
			if errors.Is(err, tg.ErrWaitTimeout) {
				return e.errorResponse(profileID, "WaitTimeout", "no reply arrived before timeout")
			}
			return e.mapTelegramUnauthorizedOr(profileID, "TelegramWaitFailed", err)
		}

		return output.Response{
			OK:      true,
			Profile: profileID,
			Data: map[string]any{
				"peer":          peer,
				"message":       message,
				"observedAtUtc": e.now(),
			},
		}
	})
}

func (e *Executor) executePressButton(ctx context.Context, profileID, peerQuery string, messageID int64, buttonIndex int, hasButtonIndex bool, buttonText string, jsonMode bool) (output.Response, bool) {
	return e.withProfileLock(profileID, jsonMode, func() output.Response {
		runtimeConfig, err := e.requireTelegramConfig()
		if err != nil {
			return e.errorResponse(profileID, "InvalidInput", err.Error())
		}

		sessionRef, err := e.authorizedSession(profileID)
		if err != nil {
			if errors.Is(err, errUnauthorizedProfile) {
				return e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized")
			}
			return e.mapStoreError(profileID, err)
		}

		peer, resp, ok := e.resolvePeer(ctx, profileID, runtimeConfig, sessionRef, peerQuery)
		if !ok {
			return resp
		}

		result, err := e.telegram.PressButton(ctx, runtimeConfig, sessionRef, tg.PressButtonRequest{
			Peer:           peer,
			MessageID:      messageID,
			ButtonIndex:    buttonIndex,
			HasButtonIndex: hasButtonIndex,
			ButtonText:     buttonText,
		})
		if err != nil {
			switch {
			case errors.Is(err, tg.ErrMessageNotFound):
				return e.errorResponse(profileID, "MessageNotFound", "message was not found")
			case errors.Is(err, tg.ErrButtonNotFound):
				return e.errorResponse(profileID, "ButtonNotFound", "button was not found")
			case errors.Is(err, tg.ErrButtonAmbiguous):
				return e.errorResponse(profileID, "ButtonAmbiguous", "button selector matched multiple buttons")
			case errors.Is(err, tg.ErrButtonUnsupported):
				return e.errorResponse(profileID, "ButtonUnsupported", "button type is not supported by this command")
			case errors.Is(err, tg.ErrButtonPasswordRequired):
				return e.errorResponse(profileID, "ButtonPasswordRequired", "button requires password verification")
			default:
				return e.mapTelegramUnauthorizedOr(profileID, "TelegramCallbackFailed", err)
			}
		}

		data := map[string]any{
			"peer":          peer,
			"action":        result.Action,
			"button":        result.Button,
			"observedAtUtc": e.now(),
		}
		if result.URL != "" {
			data["url"] = result.URL
		}
		if result.CallbackAnswer != nil {
			data["callbackAnswer"] = result.CallbackAnswer
		}

		return output.Response{
			OK:      true,
			Profile: profileID,
			Data:    data,
		}
	})
}
