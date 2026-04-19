package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/profile"
	"mi-telegram-cli/internal/tg"
)

func (e *Executor) handleAuth(ctx context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing auth subcommand"), false
	}

	switch args[0] {
	case "login":
		fs := newFlagSet("auth login")
		profileID := fs.String("profile", "", "")
		methodRaw := fs.String("method", "", "")
		phone := fs.String("phone", "", "")
		code := fs.String("code", "", "")
		password := fs.String("password", "", "")
		timeoutSeconds := fs.Int("timeout", 120, "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" {
			return e.errorResponse(*profileID, "InvalidInput", "profile is required"), *jsonMode
		}

		method, err := e.resolveLoginMethod(flagProvided(fs, "method"), *methodRaw, *phone, *code, *password, *jsonMode)
		if err != nil {
			return e.errorResponse(*profileID, "InvalidInput", err.Error()), *jsonMode
		}
		if *timeoutSeconds < 1 {
			return e.errorResponse(*profileID, "InvalidInput", "timeout must be greater than zero"), *jsonMode
		}
		if method == tg.LoginMethodQR {
			if *jsonMode {
				return e.errorResponse(*profileID, "InvalidInput", "--json is not supported with --method qr"), false
			}
			if strings.TrimSpace(*phone) != "" || strings.TrimSpace(*code) != "" || strings.TrimSpace(*password) != "" {
				return e.errorResponse(*profileID, "InvalidInput", "--method qr does not support phone, code or password flags"), false
			}
			if !e.interactive {
				return e.errorResponse(*profileID, "InvalidInput", "qr login requires an interactive terminal"), false
			}
		}

		return e.withProfileLock(*profileID, *jsonMode, func() output.Response {
			if _, err := e.store.Get(*profileID); err != nil {
				return e.mapStoreError(*profileID, err)
			}

			runtimeConfig, err := e.requireTelegramConfig()
			if err != nil {
				return e.errorResponse(*profileID, "InvalidInput", err.Error())
			}

			switch method {
			case tg.LoginMethodQR:
				return e.executeQRLogin(ctx, *profileID, runtimeConfig, time.Duration(*timeoutSeconds)*time.Second)
			default:
				return e.executeCodeLogin(ctx, *profileID, runtimeConfig, *phone, *code, *password)
			}
		})
	case "status":
		fs := newFlagSet("auth status")
		profileID := fs.String("profile", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" {
			return e.errorResponse("", "InvalidInput", "profile is required"), *jsonMode
		}

		return e.withProfileLock(*profileID, *jsonMode, func() output.Response {
			if _, err := e.store.Get(*profileID); err != nil {
				return e.mapStoreError(*profileID, err)
			}

			auth, err := e.store.LoadAuthState(*profileID)
			if err != nil {
				return e.errorResponse(*profileID, "LocalStorageFailure", err.Error())
			}

			return output.Response{
				OK:      true,
				Profile: *profileID,
				Data: map[string]any{
					"authorizationStatus": auth.AuthorizationStatus,
					"lastCheckedAtUtc":    auth.LastCheckedAtUTC,
				},
			}
		})
	case "logout":
		fs := newFlagSet("auth logout")
		profileID := fs.String("profile", "", "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if *profileID == "" {
			return e.errorResponse("", "InvalidInput", "profile is required"), *jsonMode
		}

		return e.withProfileLock(*profileID, *jsonMode, func() output.Response {
			if _, err := e.store.Get(*profileID); err != nil {
				return e.mapStoreError(*profileID, err)
			}

			removed, err := e.store.RemoveSession(*profileID)
			if err != nil {
				return e.errorResponse(*profileID, "LocalStorageFailure", err.Error())
			}

			authStatus := profile.AuthorizationUnauthorized
			if removed {
				authStatus = profile.AuthorizationLoggedOut
			}
			now := e.now()
			if err := e.store.SaveAuthState(profile.AuthState{
				ProfileID:           *profileID,
				AuthorizationStatus: authStatus,
				LastCheckedAtUTC:    &now,
				LogoutAtUTC:         &now,
			}); err != nil {
				return e.errorResponse(*profileID, "LocalStorageFailure", err.Error())
			}

			return output.Response{
				OK:      true,
				Profile: *profileID,
				Data: map[string]any{
					"authorizationStatus": authStatus,
					"sessionRemoved":      removed,
				},
			}
		})
	default:
		return e.errorResponse("", "InvalidInput", "unknown auth subcommand"), false
	}
}

func (e *Executor) resolveLoginMethod(methodProvided bool, methodRaw, phone, code, password string, jsonMode bool) (tg.LoginMethod, error) {
	method := tg.LoginMethod(strings.ToLower(strings.TrimSpace(methodRaw)))
	if methodProvided {
		return validateLoginMethod(method)
	}
	if method != "" {
		return validateLoginMethod(method)
	}
	if shouldInferCodeLogin(jsonMode, phone, code, password) || !e.interactive {
		return tg.LoginMethodCode, nil
	}
	return e.promptLoginMethod()
}

func validateLoginMethod(method tg.LoginMethod) (tg.LoginMethod, error) {
	switch method {
	case tg.LoginMethodCode, tg.LoginMethodQR:
		return method, nil
	default:
		return "", fmt.Errorf("unsupported login method %q", method)
	}
}

func shouldInferCodeLogin(jsonMode bool, phone, code, password string) bool {
	return jsonMode ||
		strings.TrimSpace(phone) != "" ||
		strings.TrimSpace(code) != "" ||
		strings.TrimSpace(password) != ""
}

func (e *Executor) promptLoginMethod() (tg.LoginMethod, error) {
	for {
		selection, err := e.prompt("Choose login method:\n1) QR\n2) Phone + code\nSelection")
		if err != nil {
			return "", fmt.Errorf("failed to read login method: %w", err)
		}

		switch strings.ToLower(strings.TrimSpace(selection)) {
		case "1", "qr":
			return tg.LoginMethodQR, nil
		case "2", "code", "phone", "phone+code", "phone + code":
			return tg.LoginMethodCode, nil
		default:
			_, _ = fmt.Fprintln(e.stderr, "Invalid selection. Enter 1, 2, qr, code or phone.")
		}
	}
}

func (e *Executor) executeCodeLogin(ctx context.Context, profileID string, runtimeConfig tg.RuntimeConfig, phone, code, password string) output.Response {
	phoneValue, err := e.promptValue("phone", phone)
	if err != nil {
		return e.errorResponse(profileID, "InvalidInput", err.Error())
	}
	if phoneValue == "" {
		return e.errorResponse(profileID, "InvalidInput", "profile and phone are required")
	}

	loginReq := tg.LoginRequest{
		Method:            tg.LoginMethodCode,
		ProfileID:         profileID,
		PhoneNumber:       phoneValue,
		VerificationCode:  strings.TrimSpace(code),
		TwoFactorPassword: strings.TrimSpace(password),
		RequestVerificationCode: func() (string, error) {
			return e.promptValue("code", code)
		},
		RequestTwoFactorPassword: func() (string, error) {
			return e.promptValue("password", password)
		},
	}

	result, err := e.telegram.Login(ctx, runtimeConfig, loginReq)
	if err != nil {
		var inputErr *tg.LoginInputError
		switch {
		case errors.As(err, &inputErr):
			return e.errorResponse(profileID, "InvalidInput", inputErr.Error())
		case errors.Is(err, tg.ErrInvalidVerificationCode):
			return e.errorResponse(profileID, "InvalidVerificationCode", err.Error())
		case errors.Is(err, tg.ErrInvalidPhoneNumber):
			return e.errorResponse(profileID, "InvalidInput", err.Error())
		default:
			return e.errorResponse(profileID, "TelegramAuthFailed", err.Error())
		}
	}

	return e.persistAuthorizedLogin(profileID, result)
}

func (e *Executor) executeQRLogin(ctx context.Context, profileID string, runtimeConfig tg.RuntimeConfig, timeout time.Duration) output.Response {
	presenter := newQRLoginPresenter(e.stderr, e.now, e.terminalSupportsANSI)
	result, err := e.telegram.Login(ctx, runtimeConfig, tg.LoginRequest{
		Method:    tg.LoginMethodQR,
		ProfileID: profileID,
		Timeout:   timeout,
		OnQRCode: func(token tg.QRLoginToken) error {
			return presenter.Show(token)
		},
	})
	if err != nil {
		switch {
		case errors.Is(err, tg.ErrAuthQRTimeout):
			return e.errorResponse(profileID, "AuthQrTimeout", err.Error())
		default:
			return e.errorResponse(profileID, "TelegramAuthFailed", err.Error())
		}
	}

	return e.persistAuthorizedLogin(profileID, result)
}

func (e *Executor) persistAuthorizedLogin(profileID string, result tg.LoginResult) output.Response {
	now := e.now()
	if err := e.store.SaveAuthState(profile.AuthState{
		ProfileID:           profileID,
		AuthorizationStatus: profile.AuthorizationAuthorized,
		AuthorizedAtUTC:     &now,
		LastCheckedAtUTC:    &now,
	}); err != nil {
		return e.errorResponse(profileID, "LocalStorageFailure", err.Error())
	}
	if err := e.store.WriteSession(profileID, result.Session); err != nil {
		return e.errorResponse(profileID, "LocalStorageFailure", err.Error())
	}

	return output.Response{
		OK:      true,
		Profile: profileID,
		Data: map[string]any{
			"authorizationStatus": profile.AuthorizationAuthorized,
			"authorizedAtUtc":     now,
			"accountSummary":      result.AccountSummary,
		},
	}
}

func (e *Executor) handleMe(ctx context.Context, args []string) (output.Response, bool) {
	fs := newFlagSet("me")
	profileID := fs.String("profile", "", "")
	jsonMode := fs.Bool("json", false, "")
	if err := fs.Parse(args); err != nil {
		return e.errorResponse("", "InvalidInput", err.Error()), true
	}
	if *profileID == "" {
		return e.errorResponse("", "InvalidInput", "profile is required"), *jsonMode
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

		accountSummary, err := e.telegram.GetMe(ctx, runtimeConfig, sessionRef)
		if err != nil {
			return e.mapTelegramUnauthorizedOr(*profileID, "TelegramMeFailed", err)
		}

		return output.Response{
			OK:      true,
			Profile: *profileID,
			Data: map[string]any{
				"accountSummary": accountSummary,
			},
		}
	})
}
