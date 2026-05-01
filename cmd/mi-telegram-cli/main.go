package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"mi-telegram-cli/internal/app"
	"mi-telegram-cli/internal/profile"
	"mi-telegram-cli/internal/tg"
)

func main() {
	if shouldPrintHelp(os.Args[1:]) {
		printUsage(os.Stdout)
		return
	}

	baseRoot, err := defaultBaseRoot()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to resolve storage root: %v\n", err)
		os.Exit(1)
	}

	store := profile.NewStore(baseRoot, time.Now().UTC)
	executor := app.NewExecutor(app.Config{
		Store:       store,
		Telegram:    tg.NewGotdClient(),
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Now:         time.Now().UTC,
		Interactive: isInteractive(os.Stdin),
		BaseRoot:    baseRoot,
		DaemonMode:  defaultDaemonMode(),
	})

	os.Exit(executor.Execute(context.Background(), os.Args[1:]))
}

func defaultDaemonMode() string {
	if value := os.Getenv("MI_TELEGRAM_CLI_DAEMON"); value != "" {
		return value
	}
	return "auto"
}

func defaultBaseRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(home, ".mi-telegram-cli"), nil
	}

	return filepath.Join(home, ".mi-telegram-cli"), nil
}

func isInteractive(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}

func shouldPrintHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}

	switch args[0] {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "mi-telegram-cli")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Commands:")
	_, _ = fmt.Fprintln(w, "  profiles add --profile <id> --display-name <name>")
	_, _ = fmt.Fprintln(w, "  profiles list")
	_, _ = fmt.Fprintln(w, "  profiles show --profile <id>")
	_, _ = fmt.Fprintln(w, "  profiles remove --profile <id> [--force]")
	_, _ = fmt.Fprintln(w, "  projects bind --root <path> --profile <id> [--create-profile] [--display-name <name>]")
	_, _ = fmt.Fprintln(w, "  projects list")
	_, _ = fmt.Fprintln(w, "  projects show --root <path>")
	_, _ = fmt.Fprintln(w, "  projects current")
	_, _ = fmt.Fprintln(w, "  projects remove --root <path>")
	_, _ = fmt.Fprintln(w, "  auth login [--profile <id>] [--method code|qr] [--phone <e164>] [--code <value>] [--password <value>] [--timeout <seconds>]")
	_, _ = fmt.Fprintln(w, "    when --method is omitted in an interactive terminal, the CLI prompts for QR or phone + code")
	_, _ = fmt.Fprintln(w, "  auth status [--profile <id>]")
	_, _ = fmt.Fprintln(w, "  auth logout [--profile <id>]")
	_, _ = fmt.Fprintln(w, "  me [--profile <id>]")
	_, _ = fmt.Fprintln(w, "  dialogs list [--profile <id>] [--query <value>] [--limit <1..100>]")
	_, _ = fmt.Fprintln(w, "  dialogs mark-read [--profile <id>] --peer <query>")
	_, _ = fmt.Fprintln(w, "  messages read [--profile <id>] --peer <query> [--limit <1..100>] [--after-id <id>]")
	_, _ = fmt.Fprintln(w, "  messages send [--profile <id>] --peer <query> --text <value>")
	_, _ = fmt.Fprintln(w, "  messages send-photo [--profile <id>] --peer <query> --file <path> [--caption <value>]")
	_, _ = fmt.Fprintln(w, "  messages wait [--profile <id>] --peer <query> [--after-id <id>] --timeout <1..300>")
	_, _ = fmt.Fprintln(w, "  messages press-button [--profile <id>] --peer <query> --message-id <id> (--button-index <n> | --button-text <value>)")
	_, _ = fmt.Fprintln(w, "  daemon start|stop|status")
	_, _ = fmt.Fprintln(w, "  audit export [--since <rfc3339>] [--profile <id>] [--operation <name>] [--errors-only]")
	_, _ = fmt.Fprintln(w, "  audit summary [--json] [--since <rfc3339>] [--profile <id>] [--operation <name>] [--errors-only]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Environment:")
	_, _ = fmt.Fprintln(w, "  MI_TELEGRAM_API_ID")
	_, _ = fmt.Fprintln(w, "  MI_TELEGRAM_API_HASH")
	_, _ = fmt.Fprintln(w, "  MI_TELEGRAM_CLI_DAEMON=auto|off|required")
	_, _ = fmt.Fprintln(w, "  MI_TELEGRAM_CLI_QUEUE_TIMEOUT_SECONDS")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Profile resolution:")
	_, _ = fmt.Fprintln(w, "  explicit --profile wins; otherwise the longest matching projects.json binding for cwd is used; fallback is qa-dev")
}
