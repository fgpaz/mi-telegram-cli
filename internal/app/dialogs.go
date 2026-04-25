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
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" || *limit < 1 || *limit > 100 {
			return e.errorResponse(*profileID, "InvalidInput", "profile is required and limit must be between 1 and 100"), *jsonMode
		}

		return e.withProfileLock(*profileID, *jsonMode, func() output.Response {
			runtimeConfig, err := e.requireTelegramConfig()
			if err != nil {
				return e.errorResponse(*profileID, "InvalidInput", err.Error())
			}

			sessionRef, err := e.authorizedSession(*profileID)
			if err != nil {
				if errors.Is(err, errUnauthorizedProfile) {
					return e.errorResponse(*profileID, "UnauthorizedProfile", "profile is not authorized")
				}
				return e.mapStoreError(*profileID, err)
			}

			items, err := e.telegram.ListDialogs(ctx, runtimeConfig, sessionRef, tg.ListDialogsRequest{
				Query: *query,
				Limit: *limit,
			})
			if err != nil {
				return e.mapTelegramUnauthorizedOr(*profileID, "TelegramListDialogsFailed", err)
			}

			return output.Response{
				OK:      true,
				Profile: *profileID,
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
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" || strings.TrimSpace(*peerQuery) == "" {
			return e.errorResponse(*profileID, "InvalidInput", "profile and peer are required"), *jsonMode
		}
		if e.isProtectedProfileForAutomation(*profileID) {
			return e.profileProtectedResponse(*profileID), *jsonMode
		}

		return e.withProfileLock(*profileID, *jsonMode, func() output.Response {
			runtimeConfig, err := e.requireTelegramConfig()
			if err != nil {
				return e.errorResponse(*profileID, "InvalidInput", err.Error())
			}

			sessionRef, err := e.authorizedSession(*profileID)
			if err != nil {
				if errors.Is(err, errUnauthorizedProfile) {
					return e.errorResponse(*profileID, "UnauthorizedProfile", "profile is not authorized")
				}
				return e.mapStoreError(*profileID, err)
			}

			peer, resp, ok := e.resolvePeer(ctx, *profileID, runtimeConfig, sessionRef, *peerQuery)
			if !ok {
				return resp
			}

			if err := e.telegram.MarkRead(ctx, runtimeConfig, sessionRef, tg.MarkReadRequest{Peer: peer}); err != nil {
				return e.mapTelegramUnauthorizedOr(*profileID, "TelegramMarkReadFailed", err)
			}

			return output.Response{
				OK:      true,
				Profile: *profileID,
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
