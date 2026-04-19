package app

import (
	"io"
	"os"
	"runtime"
	"strings"
)

func resolveTerminalSupportsANSI(w io.Writer, interactive bool, lookupEnv func(string) (string, bool), explicit *bool) bool {
	if explicit != nil {
		return *explicit
	}
	if !interactive {
		return false
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		return false
	}

	term, _ := lookupEnv("TERM")
	if strings.EqualFold(strings.TrimSpace(term), "dumb") {
		return false
	}

	if runtime.GOOS != "windows" {
		return strings.TrimSpace(term) != ""
	}

	if hasEnvValue(lookupEnv, "WT_SESSION") {
		return true
	}
	if hasEnvValue(lookupEnv, "ANSICON") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(envValue(lookupEnv, "ConEmuANSI")), "ON") {
		return true
	}
	if hasEnvValue(lookupEnv, "TERM_PROGRAM") {
		return true
	}
	if hasEnvValue(lookupEnv, "COLORTERM") {
		return true
	}
	if strings.TrimSpace(term) != "" {
		return true
	}

	return false
}

func hasEnvValue(lookupEnv func(string) (string, bool), key string) bool {
	return strings.TrimSpace(envValue(lookupEnv, key)) != ""
}

func envValue(lookupEnv func(string) (string, bool), key string) string {
	if lookupEnv == nil {
		return ""
	}
	value, ok := lookupEnv(key)
	if !ok {
		return ""
	}
	return value
}
