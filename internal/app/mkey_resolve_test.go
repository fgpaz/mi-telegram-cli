package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMkeyScopedArgvUsesProfileRefs(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "profiles", "qa-mi-herdr")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"profile":"qa-mi-herdr","MI_TELEGRAM_API_ID":"mi-telegram-cli-qa/dev/MI_TELEGRAM_API_ID","MI_TELEGRAM_API_HASH":"mi-telegram-cli-qa/dev/MI_TELEGRAM_API_HASH"}`)
	if err := os.WriteFile(filepath.Join(dir, "mkey-refs.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	argv, err := MkeyScopedArgv(root, []string{"dialogs", "list", "--profile", "qa-mi-herdr"}, func(string) string { return "" }, func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "legacy -- run-scoped mi-telegram-cli-qa dev --names MI_TELEGRAM_API_ID,MI_TELEGRAM_API_HASH --path /") {
		t.Fatalf("argv = %s", joined)
	}
	if !strings.Contains(joined, "MI_TELEGRAM_CLI_MKEY_CHILD=1") {
		t.Fatalf("missing child guard: %s", joined)
	}
}

func TestMkeyScopedArgvSkipsWhenChildOrEnvPresent(t *testing.T) {
	root := t.TempDir()
	args := []string{"me", "--profile", "qa-mi-herdr"}
	look := func(name string) (string, error) { return name, nil }
	if argv, err := MkeyScopedArgv(root, args, func(k string) string {
		if k == mkeyChildEnv {
			return "1"
		}
		return ""
	}, look); err != nil || argv != nil {
		t.Fatalf("child argv=%v err=%v", argv, err)
	}
	if argv, err := MkeyScopedArgv(root, args, func(k string) string {
		if k == "MI_TELEGRAM_API_ID" || k == "MI_TELEGRAM_API_HASH" {
			return "set"
		}
		return ""
	}, look); err != nil || argv != nil {
		t.Fatalf("env argv=%v err=%v", argv, err)
	}
	if argv, err := MkeyScopedArgv(root, []string{"profiles", "list"}, func(string) string { return "" }, look); err != nil || argv != nil {
		t.Fatalf("profiles argv=%v err=%v", argv, err)
	}
}
