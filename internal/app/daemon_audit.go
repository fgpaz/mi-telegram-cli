package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"mi-telegram-cli/internal/audit"
	"mi-telegram-cli/internal/output"
)

func (e *Executor) handleDaemon(_ context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing daemon subcommand"), false
	}
	if e.daemonManager == nil {
		return e.errorResponse("", "DaemonUnavailable", "daemon manager is not configured"), false
	}

	switch args[0] {
	case "run-internal":
		if err := e.daemonManager.Serve(); err != nil {
			return e.errorResponse("", "DaemonUnavailable", err.Error()), false
		}
		return output.Response{OK: true, Data: map[string]any{"stopped": true}}, false
	case "start":
		fs := newFlagSet("daemon start")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		state, running, err := e.daemonManager.EnsureStarted()
		if err != nil {
			return e.errorResponse("", "DaemonUnavailable", err.Error()), *jsonMode
		}
		return output.Response{
			OK: true,
			Data: map[string]any{
				"running":      running,
				"pid":          state.PID,
				"host":         state.Host,
				"port":         state.Port,
				"startedAtUtc": state.StartedAtUTC,
			},
		}, *jsonMode
	case "status":
		fs := newFlagSet("daemon status")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		state, running, err := e.daemonManager.Status()
		if err != nil {
			return e.errorResponse("", "DaemonUnavailable", err.Error()), *jsonMode
		}
		return output.Response{
			OK: true,
			Data: map[string]any{
				"running":      running,
				"pid":          state.PID,
				"host":         state.Host,
				"port":         state.Port,
				"startedAtUtc": state.StartedAtUTC,
			},
		}, *jsonMode
	case "stop":
		fs := newFlagSet("daemon stop")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		if err := e.daemonManager.Stop(); err != nil {
			return e.errorResponse("", "DaemonUnavailable", err.Error()), *jsonMode
		}
		return output.Response{OK: true, Data: map[string]any{"stopped": true}}, *jsonMode
	default:
		return e.errorResponse("", "InvalidInput", "unknown daemon subcommand"), false
	}
}

func (e *Executor) handleAudit(_ context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing audit subcommand"), false
	}
	if e.auditRecorder == nil {
		return e.errorResponse("", "LocalStorageFailure", "audit recorder is not configured"), true
	}

	switch args[0] {
	case "export":
		fs := newFlagSet("audit export")
		sinceRaw := fs.String("since", "", "")
		profileID := fs.String("profile", "", "")
		operation := fs.String("operation", "", "")
		errorsOnly := fs.Bool("errors-only", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		filter, resp, ok := e.auditFilter(*sinceRaw, *profileID, *operation, *errorsOnly)
		if !ok {
			return resp, true
		}
		if err := e.auditRecorder.Export(e.stdout, filter); err != nil {
			return e.errorResponse("", "LocalStorageFailure", err.Error()), true
		}
		return output.Response{OK: true, SuppressOutput: true}, true
	case "summary":
		fs := newFlagSet("audit summary")
		sinceRaw := fs.String("since", "", "")
		profileID := fs.String("profile", "", "")
		operation := fs.String("operation", "", "")
		errorsOnly := fs.Bool("errors-only", false, "")
		jsonMode := fs.Bool("json", false, "")
		if err := fs.Parse(args[1:]); err != nil {
			return e.errorResponse("", "InvalidInput", err.Error()), true
		}
		filter, resp, ok := e.auditFilter(*sinceRaw, *profileID, *operation, *errorsOnly)
		if !ok {
			return resp, true
		}
		summary, err := e.auditRecorder.Summarize(filter)
		if err != nil {
			return e.errorResponse("", "LocalStorageFailure", err.Error()), *jsonMode
		}
		return output.Response{OK: true, Data: map[string]any{"summary": summary}}, *jsonMode
	default:
		return e.errorResponse("", "InvalidInput", "unknown audit subcommand"), false
	}
}

func (e *Executor) auditFilter(sinceRaw, profileID, operation string, errorsOnly bool) (audit.SummaryFilter, output.Response, bool) {
	var since time.Time
	if strings.TrimSpace(sinceRaw) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(sinceRaw))
		if err != nil {
			return audit.SummaryFilter{}, e.errorResponse("", "InvalidInput", "since must be RFC3339"), false
		}
		since = parsed
	}
	return audit.SummaryFilter{
		Since:      since,
		Profile:    strings.TrimSpace(profileID),
		Operation:  strings.TrimSpace(operation),
		ErrorsOnly: errorsOnly,
	}, output.Response{}, true
}

func (e *Executor) recordAudit(args []string, resp output.Response, started time.Time, exitCode int) {
	if e.auditRecorder == nil || len(args) == 0 || args[0] == "audit" || args[0] == "daemon" {
		return
	}
	completed := e.now()
	cwd, _ := os.Getwd()
	event := audit.Event{
		EventVersion:   audit.EventVersion,
		EventID:        audit.NewEventID(),
		StartedAtUTC:   started,
		CompletedAtUTC: completed,
		Operation:      operationName(args),
		Profile:        resp.Profile,
		ProjectCwd:     cwd,
		PID:            os.Getpid(),
		QueueMs:        resp.QueueMs,
		DurationMs:     completed.Sub(started).Milliseconds(),
		OK:             resp.OK,
		ExitCode:       exitCode,
		PeerQuery:      safePeerQuery(args),
	}
	if e.daemonManager != nil {
		if state, running, err := e.daemonManager.Status(); err == nil && running {
			event.DaemonPID = state.PID
		}
	}
	if resp.Error != nil {
		event.ErrorCode = resp.Error.Code
		event.ErrorKind = errorKind(resp.Error.Code)
	}
	_ = e.auditRecorder.Append(event)
}

func operationName(args []string) string {
	if len(args) >= 2 {
		return args[0] + " " + args[1]
	}
	return args[0]
}

func safePeerQuery(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--peer" || args[i] == "--query" {
			return args[i+1]
		}
	}
	return ""
}

func errorKind(code string) string {
	switch code {
	case "InvalidInput", "FileNotFound", "UnsupportedMediaType":
		return "input"
	case "ProfileLocked", "QueueTimeout", "DaemonUnavailable", "DaemonLeaseDenied", "DaemonLeaseExpired":
		return "coordination"
	case "UnauthorizedProfile", "InvalidVerificationCode", "TelegramAuthFailed":
		return "auth"
	default:
		return "runtime"
	}
}

func writeJSONLine(w *strings.Builder, value any) {
	raw, _ := json.Marshal(value)
	_, _ = fmt.Fprintln(w, string(raw))
}
