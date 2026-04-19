package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"mi-telegram-cli/internal/app"
	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/profile"
	"mi-telegram-cli/internal/tg"
)

func TestProfilesLifecycleAndForceRemove(t *testing.T) {
	exec, store, _, stdout := newExecutor(t, nil)
	ctx := context.Background()

	if code := exec.Execute(ctx, []string{"profiles", "add", "--profile", "qa-dev", "--display-name", "QA Dev", "--json"}); code != 0 {
		t.Fatalf("Execute(add) exit code = %d, want 0", code)
	}

	addResp := decodeResponse(t, stdout.String())
	if !addResp.OK {
		t.Fatalf("profiles add ok = false, want true: %+v", addResp)
	}

	if err := store.SaveAuthState(profile.AuthState{
		ProfileID:           "qa-dev",
		AuthorizationStatus: profile.AuthorizationAuthorized,
		AuthorizedAtUTC:     ptrTime(fixedExecutorNow()),
	}); err != nil {
		t.Fatalf("SaveAuthState() error = %v", err)
	}
	if err := store.WriteSession("qa-dev", []byte("active-session")); err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"profiles", "remove", "--profile", "qa-dev", "--json"}); code == 0 {
		t.Fatalf("Execute(remove without force) exit code = %d, want non-zero", code)
	}

	removeBlocked := decodeResponse(t, stdout.String())
	if removeBlocked.Error == nil || removeBlocked.Error.Code != "ProfileDeletionBlocked" {
		t.Fatalf("profiles remove error code = %+v, want ProfileDeletionBlocked", removeBlocked.Error)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"profiles", "remove", "--profile", "qa-dev", "--force", "--json"}); code != 0 {
		t.Fatalf("Execute(remove with force) exit code = %d, want 0", code)
	}

	removeForced := decodeResponse(t, stdout.String())
	if !removeForced.OK {
		t.Fatalf("profiles remove --force ok = false, want true: %+v", removeForced)
	}

	if _, err := store.Get("qa-dev"); !errors.Is(err, profile.ErrProfileNotFound) {
		t.Fatalf("store.Get() error after remove = %v, want ErrProfileNotFound", err)
	}
}

func TestAuthLoginStatusMeAndLogout(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
		meResult: tg.AccountSummary{
			ID:          int64(42),
			Username:    "qa_dev_bot",
			DisplayName: "QA Dev",
			PhoneMasked: "+54******1234",
			IsBot:       false,
		},
	}
	exec, store, _, stdout := newExecutor(t, fake)
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--phone", "+541100000000", "--code", "12345", "--password", "pw", "--json"}); code != 0 {
		t.Fatalf("Execute(auth login) exit code = %d, want 0", code)
	}

	loginResp := decodeResponse(t, stdout.String())
	if !loginResp.OK {
		t.Fatalf("auth login ok = false, want true: %+v", loginResp)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodCode {
		t.Fatalf("auth login request method = %+v, want code", fake.loginRequests)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"auth", "status", "--profile", "qa-dev", "--json"}); code != 0 {
		t.Fatalf("Execute(auth status) exit code = %d, want 0", code)
	}

	statusResp := decodeResponse(t, stdout.String())
	if got := statusResp.Data["authorizationStatus"]; got != string(profile.AuthorizationAuthorized) {
		t.Fatalf("auth status authorizationStatus = %v, want %q", got, profile.AuthorizationAuthorized)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"me", "--profile", "qa-dev", "--json"}); code != 0 {
		t.Fatalf("Execute(me) exit code = %d, want 0", code)
	}

	meResp := decodeResponse(t, stdout.String())
	accountSummary, ok := meResp.Data["accountSummary"].(map[string]any)
	if !ok {
		t.Fatalf("me data.accountSummary missing: %+v", meResp.Data)
	}
	if _, exists := accountSummary["phone"]; exists {
		t.Fatalf("me accountSummary unexpectedly exposed raw phone: %+v", accountSummary)
	}
	if got := accountSummary["phoneMasked"]; got != "+54******1234" {
		t.Fatalf("me accountSummary.phoneMasked = %v, want %q", got, "+54******1234")
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"auth", "logout", "--profile", "qa-dev", "--json"}); code != 0 {
		t.Fatalf("Execute(auth logout) exit code = %d, want 0", code)
	}

	logoutResp := decodeResponse(t, stdout.String())
	if got := logoutResp.Data["authorizationStatus"]; got != string(profile.AuthorizationLoggedOut) {
		t.Fatalf("auth logout authorizationStatus = %v, want %q", got, profile.AuthorizationLoggedOut)
	}
}

func TestProfilesAddInitializesUnauthorizedAuthState(t *testing.T) {
	exec, store, _, stdout := newExecutor(t, nil)
	ctx := context.Background()

	if code := exec.Execute(ctx, []string{"profiles", "add", "--profile", "qa-dev", "--display-name", "QA Dev", "--json"}); code != 0 {
		t.Fatalf("Execute(add) exit code = %d, want 0", code)
	}

	resp := decodeResponse(t, stdout.String())
	if !resp.OK {
		t.Fatalf("profiles add ok = false, want true: %+v", resp)
	}

	auth, err := store.LoadAuthState("qa-dev")
	if err != nil {
		t.Fatalf("LoadAuthState() error = %v", err)
	}
	if auth.AuthorizationStatus != profile.AuthorizationUnauthorized {
		t.Fatalf("LoadAuthState() authorizationStatus = %q, want %q", auth.AuthorizationStatus, profile.AuthorizationUnauthorized)
	}
}

func TestAuthLoginReturnsTypedInvalidVerificationCode(t *testing.T) {
	fake := &fakeTelegram{
		loginErr: tg.ErrInvalidVerificationCode,
	}
	exec, store, _, stdout := newExecutor(t, fake)
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--phone", "+541100000000", "--code", "99999", "--json"}); code == 0 {
		t.Fatalf("Execute(auth login invalid code) exit code = %d, want non-zero", code)
	}

	resp := decodeResponse(t, stdout.String())
	if resp.Error == nil || resp.Error.Code != "InvalidVerificationCode" {
		t.Fatalf("auth login error = %+v, want InvalidVerificationCode", resp.Error)
	}
}

func TestAuthLoginPromptsForMethodAndRunsQRFlowWhenInteractive(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(77),
				Username:    "qa_qr_user",
				DisplayName: "QA QR",
				PhoneMasked: "+54******0000",
				IsBot:       false,
			},
			Session: []byte("session-after-qr-login"),
		},
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=ZXhhbXBsZQ==",
				ExpiresAt: fixedExecutorNow().Add(45 * time.Second),
			},
		},
	}
	var prompts []string
	responses := []string{"1"}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev"}); code != 0 {
		t.Fatalf("Execute(auth login interactive qr prompt) exit code = %d, want 0", code)
	}

	if len(prompts) != 1 || !strings.Contains(prompts[0], "Choose login method") {
		t.Fatalf("method prompts = %#v, want single method prompt", prompts)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodQR {
		t.Fatalf("auth login request method = %+v, want qr", fake.loginRequests)
	}
	if !strings.Contains(stdout.String(), "ok profile=qa-dev") {
		t.Fatalf("auth login output = %q, want final success line", stdout.String())
	}
}

func TestAuthLoginPromptsForMethodAndRunsCodeFlowWhenInteractive(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
		requireVerificationCodeCallback: true,
		expectedVerificationCode:        "12345",
	}
	var prompts []string
	responses := []string{"2", "+541100000000", "12345"}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev"}); code != 0 {
		t.Fatalf("Execute(auth login interactive code prompt) exit code = %d, want 0", code)
	}

	if got, want := prompts, []string{"Choose login method:\n1) QR\n2) Phone + code\nSelection", "phone", "code"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("prompt sequence = %#v, want %#v", got, want)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodCode {
		t.Fatalf("auth login request method = %+v, want code", fake.loginRequests)
	}
	if !strings.Contains(stdout.String(), "ok profile=qa-dev") {
		t.Fatalf("auth login output = %q, want final success line", stdout.String())
	}
}

func TestAuthLoginInteractiveCodeFlagsSkipMethodPrompt(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
		requireVerificationCodeCallback: true,
		expectedVerificationCode:        "12345",
	}
	var prompts []string
	responses := []string{"12345"}
	exec, store, _, _ := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--phone", "+541100000000"}); code != 0 {
		t.Fatalf("Execute(auth login interactive inferred code) exit code = %d, want 0", code)
	}

	if len(prompts) != 1 || prompts[0] != "code" {
		t.Fatalf("prompt sequence = %#v, want only code prompt", prompts)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodCode {
		t.Fatalf("auth login request method = %+v, want code", fake.loginRequests)
	}
}

func TestAuthLoginInteractiveJSONSkipsMethodPrompt(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
		requireVerificationCodeCallback: true,
		expectedVerificationCode:        "12345",
	}
	var prompts []string
	responses := []string{"+541100000000", "12345"}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--json"}); code != 0 {
		t.Fatalf("Execute(auth login interactive json inferred code) exit code = %d, want 0", code)
	}

	if got, want := prompts, []string{"phone", "code"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("prompt sequence = %#v, want %#v", got, want)
	}
	resp := decodeResponse(t, stdout.String())
	if !resp.OK {
		t.Fatalf("auth login interactive json ok = false, want true: %+v", resp)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodCode {
		t.Fatalf("auth login request method = %+v, want code", fake.loginRequests)
	}
}

func TestAuthLoginRepromptsOnInvalidMethodSelection(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(77),
				Username:    "qa_qr_user",
				DisplayName: "QA QR",
				PhoneMasked: "+54******0000",
				IsBot:       false,
			},
			Session: []byte("session-after-qr-login"),
		},
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=ZXhhbXBsZQ==",
				ExpiresAt: fixedExecutorNow().Add(45 * time.Second),
			},
		},
	}
	var prompts []string
	responses := []string{"wat", "qr"}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev"}); code != 0 {
		t.Fatalf("Execute(auth login invalid method then qr) exit code = %d, want 0", code)
	}

	if len(prompts) != 2 {
		t.Fatalf("method prompts = %#v, want two prompts", prompts)
	}
	if !strings.Contains(stdout.String(), "Invalid selection.") {
		t.Fatalf("auth login output = %q, want invalid selection message", stdout.String())
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodQR {
		t.Fatalf("auth login request method = %+v, want qr", fake.loginRequests)
	}
}

func TestAuthLoginExplicitMethodOverridesInteractivePrompt(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
	}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(string) (string, error) {
		return "", errors.New("prompt should not be called")
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "code", "--phone", "+541100000000", "--code", "12345", "--json"}); code != 0 {
		t.Fatalf("Execute(auth login explicit method) exit code = %d, want 0", code)
	}

	resp := decodeResponse(t, stdout.String())
	if !resp.OK {
		t.Fatalf("auth login explicit method ok = false, want true: %+v", resp)
	}
	if len(fake.loginRequests) != 1 || fake.loginRequests[0].Method != tg.LoginMethodCode {
		t.Fatalf("auth login request method = %+v, want code", fake.loginRequests)
	}
}

func TestAuthLoginPromptsForTwoFactorPasswordWhenRequired(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(42),
				Username:    "qa_dev_bot",
				DisplayName: "QA Dev",
				PhoneMasked: "+54******1234",
				IsBot:       false,
			},
			Session: []byte("session-after-login"),
		},
		requireVerificationCodeCallback:  true,
		expectedVerificationCode:         "12345",
		requireTwoFactorPasswordCallback: true,
		expectedTwoFactorPassword:        "pw-2fa",
	}
	var prompts []string
	responses := []string{"12345", "pw-2fa"}
	exec, store, _, stdout := newExecutorWithPrompt(t, fake, true, nil, func(label string) (string, error) {
		prompts = append(prompts, label)
		if len(responses) == 0 {
			return "", errors.New("unexpected prompt")
		}
		answer := responses[0]
		responses = responses[1:]
		return answer, nil
	})
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "code", "--phone", "+541100000000", "--json"}); code != 0 {
		t.Fatalf("Execute(auth login 2fa) exit code = %d, want 0", code)
	}

	if got, want := prompts, []string{"code", "password"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("prompt sequence = %#v, want %#v", got, want)
	}
	resp := decodeResponse(t, stdout.String())
	if !resp.OK {
		t.Fatalf("auth login 2fa ok = false, want true: %+v", resp)
	}
}

func TestAuthLoginQRSuccessPersistsSessionAndPrintsQRCode(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(77),
				Username:    "qa_qr_user",
				DisplayName: "QA QR",
				PhoneMasked: "+54******0000",
				IsBot:       false,
			},
			Session: []byte("session-after-qr-login"),
		},
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=ZXhhbXBsZQ==",
				ExpiresAt: fixedExecutorNow().Add(45 * time.Second),
			},
		},
	}
	exec, store, _, stdout := newExecutorWithInteractive(t, fake, true)
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--timeout", "120"}); code != 0 {
		t.Fatalf("Execute(auth login qr) exit code = %d, want 0", code)
	}

	raw := stdout.String()
	if !strings.Contains(raw, "Telegram QR Login") {
		t.Fatalf("auth login qr output missing heading: %q", raw)
	}
	if !strings.Contains(raw, "tg://login?token=ZXhhbXBsZQ==") {
		t.Fatalf("auth login qr output missing fallback link: %q", raw)
	}
	if !strings.Contains(raw, "█") && !strings.Contains(raw, "▀") && !strings.Contains(raw, "▄") {
		t.Fatalf("auth login qr output missing compact unicode QR glyphs: %q", raw)
	}
	if !strings.Contains(raw, "ok profile=qa-dev") {
		t.Fatalf("auth login qr output missing final success line: %q", raw)
	}

	auth, err := store.LoadAuthState("qa-dev")
	if err != nil {
		t.Fatalf("LoadAuthState() error = %v", err)
	}
	if auth.AuthorizationStatus != profile.AuthorizationAuthorized {
		t.Fatalf("LoadAuthState() authorizationStatus = %q, want %q", auth.AuthorizationStatus, profile.AuthorizationAuthorized)
	}

	session, err := store.ReadSession("qa-dev")
	if err != nil {
		t.Fatalf("ReadSession() error = %v", err)
	}
	if string(session) != "session-after-qr-login" {
		t.Fatalf("ReadSession() = %q, want %q", string(session), "session-after-qr-login")
	}
}

func TestAuthLoginQRReturnsTypedTimeout(t *testing.T) {
	fake := &fakeTelegram{
		loginErr: tg.ErrAuthQRTimeout,
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=dGltZW91dA==",
				ExpiresAt: fixedExecutorNow().Add(30 * time.Second),
			},
		},
	}
	exec, store, _, stdout := newExecutorWithInteractive(t, fake, true)
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--timeout", "120"}); code == 0 {
		t.Fatalf("Execute(auth login qr timeout) exit code = %d, want non-zero", code)
	}

	raw := stdout.String()
	if !strings.Contains(raw, "AuthQrTimeout") {
		t.Fatalf("auth login qr timeout output = %q, want AuthQrTimeout", raw)
	}
}

func TestAuthLoginQRRedrawsInPlaceWhenANSISupported(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(77),
				Username:    "qa_qr_user",
				DisplayName: "QA QR",
				PhoneMasked: "+54******0000",
				IsBot:       false,
			},
			Session: []byte("session-after-qr-login"),
		},
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=Zmlyc3Q=",
				ExpiresAt: fixedExecutorNow().Add(45 * time.Second),
			},
			{
				URL:       "tg://login?token=c2Vjb25k",
				ExpiresAt: fixedExecutorNow().Add(30 * time.Second),
			},
		},
	}
	exec, store, _, stdout := newExecutorWithTerminalMode(t, fake, true, boolPtr(true))
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--timeout", "120"}); code != 0 {
		t.Fatalf("Execute(auth login qr ansi redraw) exit code = %d, want 0", code)
	}

	raw := stdout.String()
	if !strings.Contains(raw, "\x1b[") {
		t.Fatalf("auth login qr ansi redraw output missing ANSI cursor controls: %q", raw)
	}
	if !strings.Contains(raw, "tg://login?token=Zmlyc3Q=") || !strings.Contains(raw, "tg://login?token=c2Vjb25k") {
		t.Fatalf("auth login qr ansi redraw output missing both QR refresh tokens: %q", raw)
	}
}

func TestAuthLoginQRFallsBackToAppendWhenANSINotSupported(t *testing.T) {
	fake := &fakeTelegram{
		loginResult: tg.LoginResult{
			AccountSummary: tg.AccountSummary{
				ID:          int64(77),
				Username:    "qa_qr_user",
				DisplayName: "QA QR",
				PhoneMasked: "+54******0000",
				IsBot:       false,
			},
			Session: []byte("session-after-qr-login"),
		},
		qrTokens: []tg.QRLoginToken{
			{
				URL:       "tg://login?token=Zmlyc3Q=",
				ExpiresAt: fixedExecutorNow().Add(45 * time.Second),
			},
			{
				URL:       "tg://login?token=c2Vjb25k",
				ExpiresAt: fixedExecutorNow().Add(30 * time.Second),
			},
		},
	}
	exec, store, _, stdout := newExecutorWithTerminalMode(t, fake, true, boolPtr(false))
	ctx := context.Background()

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--timeout", "120"}); code != 0 {
		t.Fatalf("Execute(auth login qr append fallback) exit code = %d, want 0", code)
	}

	raw := stdout.String()
	if strings.Contains(raw, "\x1b[") {
		t.Fatalf("auth login qr append fallback unexpectedly used ANSI cursor controls: %q", raw)
	}
	if strings.Count(raw, "Telegram QR Login") != 2 {
		t.Fatalf("auth login qr append fallback heading count = %d, want 2; raw=%q", strings.Count(raw, "Telegram QR Login"), raw)
	}
}

func TestAuthLoginQRRejectsInvalidFlagCombinations(t *testing.T) {
	ctx := context.Background()

	t.Run("json not supported", func(t *testing.T) {
		exec, store, _, stdout := newExecutorWithInteractive(t, &fakeTelegram{}, true)
		if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--json"}); code == 0 {
			t.Fatalf("Execute(auth login qr --json) exit code = %d, want non-zero", code)
		}
		if !strings.Contains(stdout.String(), "InvalidInput") {
			t.Fatalf("auth login qr --json output = %q, want InvalidInput", stdout.String())
		}
	})

	t.Run("phone not supported", func(t *testing.T) {
		exec, store, _, stdout := newExecutorWithInteractive(t, &fakeTelegram{}, true)
		if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if code := exec.Execute(ctx, []string{"auth", "login", "--profile", "qa-dev", "--method", "qr", "--phone", "+541100000000"}); code == 0 {
			t.Fatalf("Execute(auth login qr --phone) exit code = %d, want non-zero", code)
		}
		if !strings.Contains(stdout.String(), "InvalidInput") {
			t.Fatalf("auth login qr --phone output = %q, want InvalidInput", stdout.String())
		}
	})
}

func TestCommandsReturnProfileLockedWhenAnotherOperationOwnsLock(t *testing.T) {
	exec, store, _, stdout := newExecutor(t, nil)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	lock, err := store.AcquireLock("qa-dev")
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	if code := exec.Execute(ctx, []string{"auth", "status", "--profile", "qa-dev", "--json"}); code == 0 {
		t.Fatalf("Execute(auth status locked) exit code = %d, want non-zero", code)
	}

	resp := decodeResponse(t, stdout.String())
	if resp.Error == nil || resp.Error.Code != "ProfileLocked" {
		t.Fatalf("auth status error = %+v, want ProfileLocked", resp.Error)
	}
}

func TestDialogsAndMessagesSuccessPaths(t *testing.T) {
	fake := &fakeTelegram{
		dialogs: []tg.DialogSummary{
			{ID: int64(1), Kind: "bot", DisplayName: "multi tedi", Username: "multi_tedi_dev_bot"},
		},
		resolvePeerResult: tg.Peer{
			ID:          int64(1),
			Kind:        "bot",
			DisplayName: "multi tedi",
			Username:    "multi_tedi_dev_bot",
		},
		readMessagesResult: []tg.MessageSummary{
			{
				ID:        int64(101),
				Direction: "incoming",
				Text:      "hola",
				SentAtUTC: fixedExecutorNow(),
				Attachments: []tg.AttachmentSummary{
					{
						Kind:    "photo",
						Summary: "photo attachment",
						Details: map[string]any{"spoiler": false},
					},
				},
				Buttons: []tg.InlineButtonSummary{
					{
						Index:      0,
						Row:        0,
						Col:        0,
						Kind:       "callback",
						Text:       "Confirmar",
						Actionable: true,
					},
					{
						Index:      1,
						Row:        0,
						Col:        1,
						Kind:       "url",
						Text:       "Abrir",
						Actionable: true,
						URL:        "https://example.com",
					},
				},
			},
		},
		sendMessageResult: tg.SendResult{
			MessageID: int64(201),
			SentAtUTC: fixedExecutorNow(),
		},
		waitMessageResult: tg.MessageSummary{
			ID:        int64(301),
			Direction: "incoming",
			Text:      "respuesta",
			SentAtUTC: fixedExecutorNow(),
			Attachments: []tg.AttachmentSummary{
				{
					Kind:    "document",
					Summary: "voice note",
					Details: map[string]any{"voice": true},
				},
			},
			Buttons: []tg.InlineButtonSummary{
				{
					Index:      0,
					Row:        0,
					Col:        0,
					Kind:       "callback",
					Text:       "Siguiente",
					Actionable: true,
				},
			},
		},
	}
	exec, store, _, stdout := newExecutor(t, fake)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	if code := exec.Execute(ctx, []string{"dialogs", "list", "--profile", "qa-dev", "--query", "tedi", "--limit", "10", "--json"}); code != 0 {
		t.Fatalf("Execute(dialogs list) exit code = %d, want 0", code)
	}
	dialogsResp := decodeResponse(t, stdout.String())
	if got := dialogsResp.Data["count"]; got != float64(1) {
		t.Fatalf("dialogs list count = %v, want 1", got)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"messages", "read", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--limit", "5", "--after-id", "100", "--json"}); code != 0 {
		t.Fatalf("Execute(messages read) exit code = %d, want 0", code)
	}

	readResp := decodeResponse(t, stdout.String())
	if got := readResp.Data["count"]; got != float64(1) {
		t.Fatalf("messages read count = %v, want 1", got)
	}
	items, ok := readResp.Data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("messages read items = %+v, want one enriched message", readResp.Data["items"])
	}
	firstMessage, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("messages read item = %+v, want object", items[0])
	}
	attachments, ok := firstMessage["attachments"].([]any)
	if !ok || len(attachments) != 1 {
		t.Fatalf("messages read attachments = %+v, want one attachment", firstMessage["attachments"])
	}
	buttons, ok := firstMessage["buttons"].([]any)
	if !ok || len(buttons) != 2 {
		t.Fatalf("messages read buttons = %+v, want two inline buttons", firstMessage["buttons"])
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"messages", "send", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--text", "hola", "--json"}); code != 0 {
		t.Fatalf("Execute(messages send) exit code = %d, want 0", code)
	}

	sendResp := decodeResponse(t, stdout.String())
	if got := sendResp.Data["messageId"]; got != float64(201) {
		t.Fatalf("messages send messageId = %v, want 201", got)
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"messages", "wait", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--after-id", "200", "--timeout", "30", "--json"}); code != 0 {
		t.Fatalf("Execute(messages wait) exit code = %d, want 0", code)
	}

	waitResp := decodeResponse(t, stdout.String())
	message, ok := waitResp.Data["message"].(map[string]any)
	if !ok || message["text"] != "respuesta" {
		t.Fatalf("messages wait data.message = %+v, want text respuesta", waitResp.Data["message"])
	}
	if attachments, ok := message["attachments"].([]any); !ok || len(attachments) != 1 {
		t.Fatalf("messages wait attachments = %+v, want one attachment", message["attachments"])
	}
	if buttons, ok := message["buttons"].([]any); !ok || len(buttons) != 1 {
		t.Fatalf("messages wait buttons = %+v, want one button", message["buttons"])
	}

	stdout.Reset()
	if code := exec.Execute(ctx, []string{"dialogs", "mark-read", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--json"}); code != 0 {
		t.Fatalf("Execute(dialogs mark-read) exit code = %d, want 0", code)
	}

	markReadResp := decodeResponse(t, stdout.String())
	if got := markReadResp.Data["markedRead"]; got != true {
		t.Fatalf("dialogs mark-read markedRead = %v, want true", got)
	}
}

func TestMessagesSendWarnsAboutPossibleMSYSPathTranslationInHumanMode(t *testing.T) {
	fake := &fakeTelegram{
		resolvePeerResult: tg.Peer{
			ID:          int64(1),
			Kind:        "bot",
			DisplayName: "multi tedi",
			Username:    "multi_tedi_dev_bot",
		},
		sendMessageResult: tg.SendResult{
			MessageID: int64(201),
			SentAtUTC: fixedExecutorNow(),
		},
	}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	exec, store, _ := newExecutorWithStreams(t, fake, false, stdout, stderr)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	text := "C:/Program Files/Git/start pairing-token"
	if code := exec.Execute(ctx, []string{"messages", "send", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--text", text}); code != 0 {
		t.Fatalf("Execute(messages send human warning) exit code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), "ok profile=qa-dev") {
		t.Fatalf("stdout = %q, want human success output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "possible MSYS path translation detected") {
		t.Fatalf("stderr = %q, want MSYS warning", stderr.String())
	}
	if len(fake.sendMessageRequests) != 1 || fake.sendMessageRequests[0].Text != text {
		t.Fatalf("send message requests = %+v, want original text to pass through unchanged", fake.sendMessageRequests)
	}
}

func TestMessagesSendDoesNotWarnInJSONMode(t *testing.T) {
	fake := &fakeTelegram{
		resolvePeerResult: tg.Peer{
			ID:          int64(1),
			Kind:        "bot",
			DisplayName: "multi tedi",
			Username:    "multi_tedi_dev_bot",
		},
		sendMessageResult: tg.SendResult{
			MessageID: int64(201),
			SentAtUTC: fixedExecutorNow(),
		},
	}
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	exec, store, _ := newExecutorWithStreams(t, fake, false, stdout, stderr)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	text := "C:/Program Files/Git/start pairing-token"
	if code := exec.Execute(ctx, []string{"messages", "send", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--text", text, "--json"}); code != 0 {
		t.Fatalf("Execute(messages send json) exit code = %d, want 0", code)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want no warning in json mode", stderr.String())
	}
	resp := decodeResponse(t, stdout.String())
	if !resp.OK {
		t.Fatalf("messages send json ok = false, want true: %+v", resp)
	}
	if len(fake.sendMessageRequests) != 1 || fake.sendMessageRequests[0].Text != text {
		t.Fatalf("send message requests = %+v, want original text to pass through unchanged", fake.sendMessageRequests)
	}
}

func TestMessagesPressButtonSupportsCallbackAndURL(t *testing.T) {
	fake := &fakeTelegram{
		resolvePeerResult: tg.Peer{
			ID:          int64(1),
			Kind:        "bot",
			DisplayName: "multi tedi",
			Username:    "multi_tedi_dev_bot",
		},
		pressButtonResult: tg.PressButtonResult{
			Action: "callback",
			Button: tg.InlineButtonSummary{
				Index:      0,
				Row:        0,
				Col:        0,
				Kind:       "callback",
				Text:       "Confirmar",
				Actionable: true,
			},
			CallbackAnswer: &tg.CallbackAnswerSummary{
				Message:   "ok",
				Alert:     false,
				HasURL:    false,
				NativeUI:  false,
				CacheTime: 0,
			},
		},
	}
	exec, store, _, stdout := newExecutor(t, fake)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	if code := exec.Execute(ctx, []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--message-id", "301", "--button-index", "0", "--button-text", "Confirmar", "--json"}); code != 0 {
		t.Fatalf("Execute(messages press-button callback) exit code = %d, want 0", code)
	}

	callbackResp := decodeResponse(t, stdout.String())
	if got := callbackResp.Data["action"]; got != "callback" {
		t.Fatalf("messages press-button action = %v, want callback", got)
	}
	button, ok := callbackResp.Data["button"].(map[string]any)
	if !ok || button["text"] != "Confirmar" {
		t.Fatalf("messages press-button button = %+v, want Confirmar", callbackResp.Data["button"])
	}
	if len(fake.pressButtonRequests) != 1 || !fake.pressButtonRequests[0].HasButtonIndex || fake.pressButtonRequests[0].ButtonIndex != 0 {
		t.Fatalf("press button request = %+v, want button-index to win", fake.pressButtonRequests)
	}

	fake.pressButtonResult = tg.PressButtonResult{
		Action: "url",
		Button: tg.InlineButtonSummary{
			Index:      1,
			Row:        0,
			Col:        1,
			Kind:       "url",
			Text:       "Abrir",
			Actionable: true,
			URL:        "https://example.com",
		},
		URL: "https://example.com",
	}
	stdout.Reset()

	if code := exec.Execute(ctx, []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@multi_tedi_dev_bot", "--message-id", "301", "--button-text", "Abrir", "--json"}); code != 0 {
		t.Fatalf("Execute(messages press-button url) exit code = %d, want 0", code)
	}

	urlResp := decodeResponse(t, stdout.String())
	if got := urlResp.Data["action"]; got != "url" {
		t.Fatalf("messages press-button action = %v, want url", got)
	}
	if got := urlResp.Data["url"]; got != "https://example.com" {
		t.Fatalf("messages press-button url = %v, want https://example.com", got)
	}
}

func TestPeerAmbiguousAndWaitTimeoutReturnTypedErrors(t *testing.T) {
	fake := &fakeTelegram{
		resolvePeerErr: tg.ErrPeerAmbiguous,
		waitMessageErr: tg.ErrWaitTimeout,
	}
	exec, store, _, stdout := newExecutor(t, fake)
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	if code := exec.Execute(ctx, []string{"messages", "read", "--profile", "qa-dev", "--peer", "tedi", "--json"}); code == 0 {
		t.Fatalf("Execute(messages read ambiguous) exit code = %d, want non-zero", code)
	}

	readResp := decodeResponse(t, stdout.String())
	if readResp.Error == nil || readResp.Error.Code != "PeerAmbiguous" {
		t.Fatalf("messages read error = %+v, want PeerAmbiguous", readResp.Error)
	}

	fake.resolvePeerErr = nil
	fake.resolvePeerResult = tg.Peer{ID: int64(1), Kind: "bot", DisplayName: "bot", Username: "bot"}
	stdout.Reset()

	if code := exec.Execute(ctx, []string{"messages", "wait", "--profile", "qa-dev", "--peer", "@bot", "--timeout", "5", "--json"}); code == 0 {
		t.Fatalf("Execute(messages wait timeout) exit code = %d, want non-zero", code)
	}

	waitResp := decodeResponse(t, stdout.String())
	if waitResp.Error == nil || waitResp.Error.Code != "WaitTimeout" {
		t.Fatalf("messages wait error = %+v, want WaitTimeout", waitResp.Error)
	}
}

func TestMessagesPressButtonReturnsTypedErrors(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		args      []string
		wantCode  string
		wantExit0 bool
	}{
		{
			name:     "ambiguous selector",
			err:      tg.ErrButtonAmbiguous,
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "301", "--button-text", "Elegir", "--json"},
			wantCode: "ButtonAmbiguous",
		},
		{
			name:     "button not found",
			err:      tg.ErrButtonNotFound,
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "301", "--button-index", "4", "--json"},
			wantCode: "ButtonNotFound",
		},
		{
			name:     "button unsupported",
			err:      tg.ErrButtonUnsupported,
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "301", "--button-index", "0", "--json"},
			wantCode: "ButtonUnsupported",
		},
		{
			name:     "message not found",
			err:      tg.ErrMessageNotFound,
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "999", "--button-index", "0", "--json"},
			wantCode: "MessageNotFound",
		},
		{
			name:     "password required",
			err:      tg.ErrButtonPasswordRequired,
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "301", "--button-index", "0", "--json"},
			wantCode: "ButtonPasswordRequired",
		},
		{
			name:     "callback failure",
			err:      errors.New("rpc failed"),
			args:     []string{"messages", "press-button", "--profile", "qa-dev", "--peer", "@bot", "--message-id", "301", "--button-index", "0", "--json"},
			wantCode: "TelegramCallbackFailed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeTelegram{
				resolvePeerResult: tg.Peer{ID: int64(1), Kind: "bot", DisplayName: "bot", Username: "bot"},
				pressButtonErr:    tc.err,
			}
			exec, store, _, stdout := newExecutor(t, fake)
			ctx := context.Background()
			createAuthorizedProfile(t, store, "qa-dev")

			if code := exec.Execute(ctx, tc.args); code == 0 {
				t.Fatalf("Execute(%s) exit code = %d, want non-zero", tc.name, code)
			}

			resp := decodeResponse(t, stdout.String())
			if resp.Error == nil || resp.Error.Code != tc.wantCode {
				t.Fatalf("messages press-button error = %+v, want %s", resp.Error, tc.wantCode)
			}
		})
	}
}

func TestMeRequiresTelegramRuntimeConfig(t *testing.T) {
	exec, store, env, stdout := newExecutor(t, &fakeTelegram{})
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")
	delete(env, "MI_TELEGRAM_API_ID")

	if code := exec.Execute(ctx, []string{"me", "--profile", "qa-dev", "--json"}); code == 0 {
		t.Fatalf("Execute(me without runtime config) exit code = %d, want non-zero", code)
	}

	resp := decodeResponse(t, stdout.String())
	if resp.Error == nil || resp.Error.Code != "InvalidInput" {
		t.Fatalf("me error = %+v, want InvalidInput", resp.Error)
	}
}

func TestMeMapsTelegramUnauthorizedToUnauthorizedProfile(t *testing.T) {
	exec, store, _, stdout := newExecutor(t, &fakeTelegram{meErr: tg.ErrUnauthorized})
	ctx := context.Background()
	createAuthorizedProfile(t, store, "qa-dev")

	if code := exec.Execute(ctx, []string{"me", "--profile", "qa-dev", "--json"}); code == 0 {
		t.Fatalf("Execute(me unauthorized) exit code = %d, want non-zero", code)
	}

	resp := decodeResponse(t, stdout.String())
	if resp.Error == nil || resp.Error.Code != "UnauthorizedProfile" {
		t.Fatalf("me error = %+v, want UnauthorizedProfile", resp.Error)
	}
}

type fakeTelegram struct {
	loginResult                      tg.LoginResult
	loginErr                         error
	qrTokens                         []tg.QRLoginToken
	loginRequests                    []tg.LoginRequest
	requireVerificationCodeCallback  bool
	expectedVerificationCode         string
	requireTwoFactorPasswordCallback bool
	expectedTwoFactorPassword        string
	meResult                         tg.AccountSummary
	meErr                            error
	dialogs                          []tg.DialogSummary
	listDialogsErr                   error
	resolvePeerResult                tg.Peer
	resolvePeerErr                   error
	readMessagesResult               []tg.MessageSummary
	readMessagesErr                  error
	sendMessageResult                tg.SendResult
	sendMessageErr                   error
	sendMessageRequests              []tg.SendMessageRequest
	waitMessageResult                tg.MessageSummary
	waitMessageErr                   error
	pressButtonResult                tg.PressButtonResult
	pressButtonErr                   error
	pressButtonRequests              []tg.PressButtonRequest
	markReadErr                      error
}

func (f *fakeTelegram) Login(_ context.Context, _ tg.RuntimeConfig, req tg.LoginRequest) (tg.LoginResult, error) {
	f.loginRequests = append(f.loginRequests, req)
	if req.Method == tg.LoginMethodCode {
		if f.requireVerificationCodeCallback {
			if strings.TrimSpace(req.VerificationCode) != "" {
				return tg.LoginResult{}, &tg.LoginInputError{Field: "code", Message: "verification code should be requested after send-code"}
			}
			if req.RequestVerificationCode == nil {
				return tg.LoginResult{}, &tg.LoginInputError{Field: "code", Message: "missing verification code callback"}
			}
			code, err := req.RequestVerificationCode()
			if err != nil {
				return tg.LoginResult{}, err
			}
			if f.expectedVerificationCode != "" && code != f.expectedVerificationCode {
				return tg.LoginResult{}, tg.ErrInvalidVerificationCode
			}
		}

		if f.requireTwoFactorPasswordCallback {
			if strings.TrimSpace(req.TwoFactorPassword) != "" {
				return tg.LoginResult{}, &tg.LoginInputError{Field: "password", Message: "two factor password should be requested lazily"}
			}
			if req.RequestTwoFactorPassword == nil {
				return tg.LoginResult{}, &tg.LoginInputError{Field: "password", Message: "missing two factor password callback"}
			}
			password, err := req.RequestTwoFactorPassword()
			if err != nil {
				return tg.LoginResult{}, err
			}
			if f.expectedTwoFactorPassword != "" && password != f.expectedTwoFactorPassword {
				return tg.LoginResult{}, tg.ErrTwoFactorRequired
			}
		}
	}
	for _, token := range f.qrTokens {
		if req.OnQRCode == nil {
			continue
		}
		if err := req.OnQRCode(token); err != nil {
			return tg.LoginResult{}, err
		}
	}
	return f.loginResult, f.loginErr
}

func (f *fakeTelegram) GetMe(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef) (tg.AccountSummary, error) {
	return f.meResult, f.meErr
}

func (f *fakeTelegram) ListDialogs(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, _ tg.ListDialogsRequest) ([]tg.DialogSummary, error) {
	return f.dialogs, f.listDialogsErr
}

func (f *fakeTelegram) ResolvePeer(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, _ tg.ResolvePeerRequest) (tg.Peer, error) {
	return f.resolvePeerResult, f.resolvePeerErr
}

func (f *fakeTelegram) ReadMessages(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, _ tg.ReadMessagesRequest) ([]tg.MessageSummary, error) {
	return f.readMessagesResult, f.readMessagesErr
}

func (f *fakeTelegram) SendMessage(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, req tg.SendMessageRequest) (tg.SendResult, error) {
	f.sendMessageRequests = append(f.sendMessageRequests, req)
	return f.sendMessageResult, f.sendMessageErr
}

func (f *fakeTelegram) WaitMessage(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, _ tg.WaitMessageRequest) (tg.MessageSummary, error) {
	return f.waitMessageResult, f.waitMessageErr
}

func (f *fakeTelegram) PressButton(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, req tg.PressButtonRequest) (tg.PressButtonResult, error) {
	f.pressButtonRequests = append(f.pressButtonRequests, req)
	return f.pressButtonResult, f.pressButtonErr
}

func (f *fakeTelegram) MarkRead(_ context.Context, _ tg.RuntimeConfig, _ tg.SessionRef, _ tg.MarkReadRequest) error {
	return f.markReadErr
}

func newExecutor(t *testing.T, fake *fakeTelegram) (*app.Executor, *profile.Store, map[string]string, *strings.Builder) {
	return newExecutorWithInteractive(t, fake, false)
}

func newExecutorWithInteractive(t *testing.T, fake *fakeTelegram, interactive bool) (*app.Executor, *profile.Store, map[string]string, *strings.Builder) {
	return newExecutorWithPrompt(t, fake, interactive, nil, nil)
}

func newExecutorWithStreams(t *testing.T, fake *fakeTelegram, interactive bool, stdout, stderr *strings.Builder) (*app.Executor, *profile.Store, map[string]string) {
	t.Helper()

	if fake == nil {
		fake = &fakeTelegram{}
	}
	if stdout == nil {
		stdout = &strings.Builder{}
	}
	if stderr == nil {
		stderr = stdout
	}

	store := profile.NewStore(t.TempDir(), fixedExecutorNow)
	env := map[string]string{
		"MI_TELEGRAM_API_ID":   "12345",
		"MI_TELEGRAM_API_HASH": "secret-hash",
	}

	exec := app.NewExecutor(app.Config{
		Store:       store,
		Telegram:    fake,
		Stdout:      stdout,
		Stderr:      stderr,
		Now:         fixedExecutorNow,
		Interactive: interactive,
		LookupEnv: func(key string) (string, bool) {
			v, ok := env[key]
			return v, ok
		},
	})

	return exec, store, env
}

func newExecutorWithTerminalMode(t *testing.T, fake *fakeTelegram, interactive bool, terminalSupportsANSI *bool) (*app.Executor, *profile.Store, map[string]string, *strings.Builder) {
	return newExecutorWithPrompt(t, fake, interactive, terminalSupportsANSI, nil)
}

func newExecutorWithPrompt(t *testing.T, fake *fakeTelegram, interactive bool, terminalSupportsANSI *bool, prompt func(string) (string, error)) (*app.Executor, *profile.Store, map[string]string, *strings.Builder) {
	t.Helper()

	if fake == nil {
		fake = &fakeTelegram{}
	}

	store := profile.NewStore(t.TempDir(), fixedExecutorNow)
	stdout := &strings.Builder{}
	env := map[string]string{
		"MI_TELEGRAM_API_ID":   "12345",
		"MI_TELEGRAM_API_HASH": "secret-hash",
	}

	exec := app.NewExecutor(app.Config{
		Store:                store,
		Telegram:             fake,
		Stdout:               stdout,
		Stderr:               stdout,
		Now:                  fixedExecutorNow,
		Prompt:               prompt,
		Interactive:          interactive,
		TerminalSupportsANSI: terminalSupportsANSI,
		LookupEnv: func(key string) (string, bool) {
			v, ok := env[key]
			return v, ok
		},
	})

	return exec, store, env, stdout
}

func createAuthorizedProfile(t *testing.T, store *profile.Store, profileID string) {
	t.Helper()

	if _, err := store.Create(profileID, "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.SaveAuthState(profile.AuthState{
		ProfileID:           profileID,
		AuthorizationStatus: profile.AuthorizationAuthorized,
		AuthorizedAtUTC:     ptrTime(fixedExecutorNow()),
		LastCheckedAtUTC:    ptrTime(fixedExecutorNow()),
	}); err != nil {
		t.Fatalf("SaveAuthState() error = %v", err)
	}

	if err := store.WriteSession(profileID, []byte("session")); err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}
}

func decodeResponse(t *testing.T, raw string) output.Response {
	t.Helper()

	var resp output.Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", raw, err)
	}
	return resp
}

func fixedExecutorNow() time.Time {
	return time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
