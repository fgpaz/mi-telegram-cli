package app

import (
	"context"
	"strings"

	"mi-telegram-cli/internal/output"
)

func (e *Executor) handleProfiles(_ context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing profiles subcommand"), false
	}

	switch args[0] {
	case "add":
		fs := newFlagSet("profiles add")
		profileID := fs.String("profile", "", "")
		displayName := fs.String("display-name", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" || strings.TrimSpace(*displayName) == "" {
			return e.errorResponse(*profileID, "InvalidInput", "profile and display-name are required"), *jsonMode
		}

		created, err := e.store.Create(*profileID, strings.TrimSpace(*displayName), "")
		if err != nil {
			return e.mapStoreError(*profileID, err), *jsonMode
		}

		return output.Response{
			OK:      true,
			Profile: created.ID,
			Data: map[string]any{
				"profileId":   created.ID,
				"displayName": created.DisplayName,
				"storageRoot": created.StorageRoot,
				"status":      created.Status,
			},
		}, *jsonMode
	case "list":
		fs := newFlagSet("profiles list")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}

		items, err := e.store.List()
		if err != nil {
			return e.errorResponse("", "LocalStorageFailure", err.Error()), *jsonMode
		}

		return output.Response{
			OK:      true,
			Profile: "",
			Data: map[string]any{
				"items": items,
			},
		}, *jsonMode
	case "show":
		fs := newFlagSet("profiles show")
		profileID := fs.String("profile", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" {
			return e.errorResponse("", "InvalidInput", "profile is required"), *jsonMode
		}

		view, err := e.store.Get(*profileID)
		if err != nil {
			return e.mapStoreError(*profileID, err), *jsonMode
		}

		return output.Response{
			OK:      true,
			Profile: view.ID,
			Data: map[string]any{
				"profileId":           view.ID,
				"displayName":         view.DisplayName,
				"storageRoot":         view.StorageRoot,
				"status":              view.Status,
				"authorizationStatus": view.AuthorizationStatus,
			},
		}, *jsonMode
	case "remove":
		fs := newFlagSet("profiles remove")
		profileID := fs.String("profile", "", "")
		force := fs.Bool("force", false, "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" {
			return e.errorResponse("", "InvalidInput", "profile is required"), *jsonMode
		}

		lock, err := e.store.AcquireLock(*profileID)
		if err != nil {
			return e.mapStoreError(*profileID, err), *jsonMode
		}
		defer func() { _ = lock.Release() }()

		auth, err := e.store.LoadAuthState(*profileID)
		if err != nil {
			return e.mapStoreError(*profileID, err), *jsonMode
		}

		hasSession, err := e.store.SessionExists(*profileID)
		if err != nil {
			return e.errorResponse(*profileID, "LocalStorageFailure", err.Error()), *jsonMode
		}

		if !*force && (auth.AuthorizationStatus == "Authorized" || hasSession) {
			return e.errorResponse(*profileID, "ProfileDeletionBlocked", "profile has an active session"), *jsonMode
		}

		if err := e.store.Delete(*profileID); err != nil {
			return e.errorResponse(*profileID, "LocalStorageFailure", err.Error()), *jsonMode
		}

		return output.Response{
			OK:      true,
			Profile: *profileID,
			Data: map[string]any{
				"removed":            true,
				"storageRootDeleted": true,
			},
		}, *jsonMode
	default:
		return e.errorResponse("", "InvalidInput", "unknown profiles subcommand"), false
	}
}
