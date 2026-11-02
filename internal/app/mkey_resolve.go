package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const mkeyChildEnv = "MI_TELEGRAM_CLI_MKEY_CHILD"

// mkeyRefs names the two external Telegram API credentials for one profile.
// Values are mkey refs project/env/name, never secret material.
type mkeyRefs struct {
	Profile string `json:"profile"`
	APIID   string `json:"MI_TELEGRAM_API_ID"`
	APIHash string `json:"MI_TELEGRAM_API_HASH"`
}

// MkeyScopedArgv builds `mkey legacy run-scoped ...` when the profile has
// refs and the process does not already have the API env. A nil argv means
// the caller should continue as-is.
func MkeyScopedArgv(baseRoot string, args []string, getenv func(string) string, lookPath func(string) (string, error)) ([]string, error) {
	if getenv == nil || strings.TrimSpace(getenv(mkeyChildEnv)) == "1" {
		return nil, nil
	}
	if strings.TrimSpace(getenv("MI_TELEGRAM_API_ID")) != "" && strings.TrimSpace(getenv("MI_TELEGRAM_API_HASH")) != "" {
		return nil, nil
	}
	if !commandNeedsTelegramAPI(args) {
		return nil, nil
	}
	profileID := explicitProfile(args)
	if profileID == "" {
		profileID = "qa-dev"
	}
	raw, err := os.ReadFile(filepath.Join(baseRoot, "profiles", profileID, "mkey-refs.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var refs mkeyRefs
	if err := json.Unmarshal(raw, &refs); err != nil {
		return nil, err
	}
	project, env, names, err := scopedNames(refs)
	if err != nil || project == "" {
		return nil, err
	}
	mkey, err := lookPath("mkey")
	if err != nil {
		return nil, err
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	argv := []string{
		mkey, "legacy", "--", "run-scoped", project, env,
		"--names", strings.Join(names, ","),
		"--path", "/",
		"--",
		"env", mkeyChildEnv + "=1",
		exe,
	}
	return append(argv, args...), nil
}

func commandNeedsTelegramAPI(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "auth", "me", "dialogs", "messages", "daemon":
		return true
	default:
		return false
	}
}

func explicitProfile(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--profile" {
			return strings.TrimSpace(args[i+1])
		}
	}
	return ""
}

func scopedNames(refs mkeyRefs) (string, string, []string, error) {
	idProject, idEnv, idName, ok := splitRef(refs.APIID)
	if !ok || idName != "MI_TELEGRAM_API_ID" {
		return "", "", nil, nil
	}
	hashProject, hashEnv, hashName, ok := splitRef(refs.APIHash)
	if !ok || hashName != "MI_TELEGRAM_API_HASH" {
		return "", "", nil, nil
	}
	if idProject != hashProject || idEnv != hashEnv {
		return "", "", nil, nil
	}
	return idProject, idEnv, []string{"MI_TELEGRAM_API_ID", "MI_TELEGRAM_API_HASH"}, nil
}

func splitRef(ref string) (string, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(ref), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
