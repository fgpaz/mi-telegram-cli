package app

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/profile"
)

var qaProfileIDPattern = regexp.MustCompile(`^qa-[A-Za-z0-9][A-Za-z0-9._-]*$`)

func (e *Executor) handleProjects(_ context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing projects subcommand"), false
	}

	switch args[0] {
	case "bind":
		fs := newFlagSet("projects bind")
		root := fs.String("root", "", "")
		profileID := fs.String("profile", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		resolvedProfileID := strings.TrimSpace(*profileID)
		if strings.TrimSpace(*root) == "" || resolvedProfileID == "" {
			return e.errorResponse(resolvedProfileID, "InvalidInput", "root and profile are required"), *jsonMode
		}
		if e.isProtectedProfileForAutomation(resolvedProfileID) {
			return e.profileProtectedResponse(resolvedProfileID), *jsonMode
		}
		if !qaProfileIDPattern.MatchString(resolvedProfileID) {
			return e.errorResponse(resolvedProfileID, "ProfileNotEligibleForBinding", "project binding requires an existing authorized QA profile with a local session"), *jsonMode
		}
		view, err := e.store.Get(resolvedProfileID)
		if err != nil {
			return e.mapStoreError(resolvedProfileID, err), *jsonMode
		}
		hasSession, err := e.store.SessionExists(resolvedProfileID)
		if err != nil {
			return e.mapStoreError(resolvedProfileID, err), *jsonMode
		}
		if view.AuthorizationStatus != profile.AuthorizationAuthorized || !hasSession {
			return e.errorResponse(resolvedProfileID, "ProfileNotEligibleForBinding", "project binding requires an existing authorized QA profile with a local session"), *jsonMode
		}

		binding, err := e.store.BindProject(*root, resolvedProfileID, "")
		if err != nil {
			return e.errorResponse(resolvedProfileID, "LocalStorageFailure", err.Error()), *jsonMode
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
