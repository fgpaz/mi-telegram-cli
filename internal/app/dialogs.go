package app

import (
	"context"
	"errors"
	"strings"

	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/tg"
)

func (e *Executor) handleDialogs(ctx context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing dialogs subcommand"), false
	}

	switch args[0] {
	case "list":
		fs := newFlagSet("dialogs list")
		profileID := fs.String("profile", "", "")
		query := fs.String("query", "", "")
		limit := fs.Int("limit", 20, "")
		jsonMode := fs.Bool("json", false, "")
		queueTimeoutSeconds := queueTimeoutFlag(fs, e.defaultQueueTimeout())
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		resolvedProfileID, err := e.resolveEffectiveProfile(*profileID, flagProvided(fs, "profile"))
		if err != nil {
			return e.mapStoreError(resolvedProfileID, err), *jsonMode
		}
		if resolvedProfileID == "" || *limit < 1 || *limit > 100 {
			return e.errorResponse(resolvedProfileID, "InvalidInput", "profile is required and limit must be between 1 and 100"), *jsonMode
		}

		queueTimeout := durationFromSeconds(*queueTimeoutSeconds)
		if queueTimeout < 0 {
			return e.errorResponse(resolvedProfileID, "InvalidInput", "queue-timeout must be zero or greater"), *jsonMode
		}
		return e.withProfileLock(resolvedProfileID, *jsonMode, queueTimeout, func() output.Response {
			runtimeConfig, err := e.requireTelegramConfig()
			if err != nil {
				return e.errorResponse(resolvedProfileID, "InvalidInput", err.Error())
			}

			sessionRef, err := e.authorizedSession(resolvedProfileID)
			if err != nil {
				if errors.Is(err, errUnauthorizedProfile) {
					return e.errorResponse(resolvedProfileID, "UnauthorizedProfile", "profile is not authorized")
				}
				return e.mapStoreError(resolvedProfileID, err)
			}

			items, err := e.telegram.ListDialogs(ctx, runtimeConfig, sessionRef, tg.ListDialogsRequest{
				Query: *query,
				Limit: *limit,
			})
			if err != nil {
				return e.mapTelegramUnauthorizedOr(resolvedProfileID, "TelegramListDialogsFailed", err)
			}

			return output.Response{
				OK:      true,
				Profile: resolvedProfileID,
				Data: map[string]any{
					"items": items,
					"count": len(items),
				},
			}
		})
	case "mark-read":
		fs := newFlagSet("dialogs mark-read")
		profileID := fs.String("profile", "", "")
		peerQuery := fs.String("peer", "", "")
		jsonMode := fs.Bool("json", false, "")
		queueTimeoutSeconds := queueTimeoutFlag(fs, e.defaultQueueTimeout())
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		resolvedProfileID, err := e.resolveEffectiveProfile(*profileID, flagProvided(fs, "profile"))
		if err != nil {
			return e.mapStoreError(resolvedProfileID, err), *jsonMode
		}
		if resolvedProfileID == "" || strings.TrimSpace(*peerQuery) == "" {
			return e.errorResponse(resolvedProfileID, "InvalidInput", "profile and peer are required"), *jsonMode
		}
		if e.isProtectedProfileForAutomation(resolvedProfileID) {
			return e.profileProtectedResponse(resolvedProfileID), *jsonMode
		}

		queueTimeout := durationFromSeconds(*queueTimeoutSeconds)
		if queueTimeout < 0 {
			return e.errorResponse(resolvedProfileID, "InvalidInput", "queue-timeout must be zero or greater"), *jsonMode
		}
		return e.withProfileLock(resolvedProfileID, *jsonMode, queueTimeout, func() output.Response {
			runtimeConfig, err := e.requireTelegramConfig()
			if err != nil {
				return e.errorResponse(resolvedProfileID, "InvalidInput", err.Error())
			}

			sessionRef, err := e.authorizedSession(resolvedProfileID)
			if err != nil {
				if errors.Is(err, errUnauthorizedProfile) {
					return e.errorResponse(resolvedProfileID, "UnauthorizedProfile", "profile is not authorized")
				}
				return e.mapStoreError(resolvedProfileID, err)
			}

			peer, resp, ok := e.resolvePeer(ctx, resolvedProfileID, runtimeConfig, sessionRef, *peerQuery)
			if !ok {
				return resp
			}

			if err := e.telegram.MarkRead(ctx, runtimeConfig, sessionRef, tg.MarkReadRequest{Peer: peer}); err != nil {
				return e.mapTelegramUnauthorizedOr(resolvedProfileID, "TelegramMarkReadFailed", err)
			}

			return output.Response{
				OK:      true,
				Profile: resolvedProfileID,
				Data: map[string]any{
					"peer":           peer,
					"markedRead":     true,
					"completedAtUtc": e.now(),
				},
			}
		})
	default:
		return e.errorResponse("", "InvalidInput", "unknown dialogs subcommand"), false
	}
}
