package app

import (
	"context"
	"errors"
	"strings"

	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/profile"
)

func (e *Executor) handleProjects(_ context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing projects subcommand"), false
	}

	switch args[0] {
	case "bind":
		fs := newFlagSet("projects bind")
		root := fs.String("root", "", "")
		profileID := fs.String("profile", "", "")
		createProfile := fs.Bool("create-profile", false, "")
		displayName := fs.String("display-name", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if strings.TrimSpace(*root) == "" || strings.TrimSpace(*profileID) == "" {
			return e.errorResponse(*profileID, "InvalidInput", "root and profile are required"), *jsonMode
		}

		if _, err := e.store.Get(*profileID); err != nil {
			if !errors.Is(err, profile.ErrProfileNotFound) || !*createProfile {
				return e.mapStoreError(*profileID, err), *jsonMode
			}
			name := strings.TrimSpace(*displayName)
			if name == "" {
				name = *profileID
			}
			if _, err := e.store.Create(*profileID, name, ""); err != nil {
				return e.mapStoreError(*profileID, err), *jsonMode
			}
		}

		binding, err := e.store.BindProject(*root, *profileID, *displayName)
		if err != nil {
			return e.errorResponse(*profileID, "LocalStorageFailure", err.Error()), *jsonMode
		}
		return output.Response{
			OK:      true,
			Profile: binding.ProfileID,
			Data: map[string]any{
				"binding": binding,
			},
		}, *jsonMode
	case "list":
		fs := newFlagSet("projects list")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		items, err := e.store.ListProjectBindings()
		if err != nil {
			return e.errorResponse("", "LocalStorageFailure", err.Error()), *jsonMode
		}
		return output.Response{OK: true, Data: map[string]any{"items": items, "count": len(items)}}, *jsonMode
	case "show":
		fs := newFlagSet("projects show")
		root := fs.String("root", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if strings.TrimSpace(*root) == "" {
			return e.errorResponse("", "InvalidInput", "root is required"), *jsonMode
		}
		binding, err := e.store.GetProjectBinding(*root)
		if err != nil {
			return e.mapProjectError(err), *jsonMode
		}
		return output.Response{OK: true, Profile: binding.ProfileID, Data: map[string]any{"binding": binding}}, *jsonMode
	case "current":
		fs := newFlagSet("projects current")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		binding, ok, err := e.store.ResolveProjectBinding(e.currentCwd())
		if err != nil {
			return e.errorResponse("", "LocalStorageFailure", err.Error()), *jsonMode
		}
		if !ok {
			return output.Response{OK: true, Profile: "qa-dev", Data: map[string]any{
				"matched":   false,
				"profileId": "qa-dev",
				"cwd":       e.currentCwd(),
				"fallback":  true,
			}}, *jsonMode
		}
		return output.Response{OK: true, Profile: binding.ProfileID, Data: map[string]any{
			"matched": true,
			"binding": binding,
			"cwd":     e.currentCwd(),
		}}, *jsonMode
	case "remove":
		fs := newFlagSet("projects remove")
		root := fs.String("root", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if strings.TrimSpace(*root) == "" {
			return e.errorResponse("", "InvalidInput", "root is required"), *jsonMode
		}
		removed, err := e.store.RemoveProjectBinding(*root)
		if err != nil {
			return e.mapProjectError(err), *jsonMode
		}
		return output.Response{OK: true, Profile: removed.ProfileID, Data: map[string]any{
			"removed": true,
			"binding": removed,
		}}, *jsonMode
	default:
		return e.errorResponse("", "InvalidInput", "unknown projects subcommand"), false
	}
}

func (e *Executor) mapProjectError(err error) output.Response {
	if errors.Is(err, profile.ErrProjectBindingNotFound) {
		return e.errorResponse("", "ProjectBindingNotFound", err.Error())
	}
	return e.errorResponse("", "LocalStorageFailure", err.Error())
}
