package app

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"mi-telegram-cli/internal/audit"
	"mi-telegram-cli/internal/daemon"
	"mi-telegram-cli/internal/output"
	"mi-telegram-cli/internal/profile"
	"mi-telegram-cli/internal/tg"
)

type Config struct {
	Store                *profile.Store
	Telegram             tg.Client
	Stdin                io.Reader
	Stdout               io.Writer
	Stderr               io.Writer
	Now                  func() time.Time
	LookupEnv            func(string) (string, bool)
	Prompt               func(string) (string, error)
	Interactive          bool
	TerminalSupportsANSI *bool
	BaseRoot             string
	DaemonMode           string
	Cwd                  string
}

type Executor struct {
	store                *profile.Store
	telegram             tg.Client
	stdin                io.Reader
	stdout               io.Writer
	stderr               io.Writer
	now                  func() time.Time
	lookupEnv            func(string) (string, bool)
	prompt               func(string) (string, error)
	interactive          bool
	terminalSupportsANSI bool
	auditRecorder        *audit.Recorder
	daemonManager        *daemon.Manager
	daemonMode           string
	cwd                  string
}

var errUnauthorizedProfile = errors.New("unauthorized profile")
var errProjectProfileMissing = errors.New("project profile missing")

var protectedAutomationProfiles = map[string]bool{
	"qa-alt": true,
}

const profileProtectedMessage = "qa-alt is protected real-user state; use qa-dev for automation"

func NewExecutor(cfg Config) *Executor {
	now := cfg.Now
	if now == nil {
		now = time.Now().UTC
	}

	lookupEnv := cfg.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stdin := cfg.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	prompt := cfg.Prompt
	if prompt == nil {
		reader := bufio.NewReader(stdin)
		prompt = func(label string) (string, error) {
			if _, err := fmt.Fprintf(stderr, "%s: ", label); err != nil {
				return "", err
			}
			value, err := reader.ReadString('\n')
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(value), nil
		}
	}

	executor := &Executor{
		store:                cfg.Store,
		telegram:             cfg.Telegram,
		stdin:                stdin,
		stdout:               stdout,
		stderr:               stderr,
		now:                  now,
		lookupEnv:            lookupEnv,
		prompt:               prompt,
		interactive:          cfg.Interactive,
		terminalSupportsANSI: resolveTerminalSupportsANSI(stderr, cfg.Interactive, lookupEnv, cfg.TerminalSupportsANSI),
		daemonMode:           strings.TrimSpace(cfg.DaemonMode),
		cwd:                  strings.TrimSpace(cfg.Cwd),
	}
	if cfg.BaseRoot != "" {
		executor.auditRecorder = audit.NewRecorder(cfg.BaseRoot, now)
		executor.daemonManager = daemon.NewManager(cfg.BaseRoot, now)
	}
	return executor
}

func (e *Executor) Execute(ctx context.Context, args []string) int {
	started := e.now()
	resp, jsonMode := e.dispatch(ctx, args)
	if !resp.SuppressOutput {
		if jsonMode {
			_ = output.WriteJSON(e.stdout, resp)
		} else {
			_ = output.WriteHuman(e.stdout, resp)
		}
	}

	if resp.OK {
		e.recordAudit(args, resp, started, 0)
		return 0
	}
	e.recordAudit(args, resp, started, 1)
	return 1
}

func (e *Executor) dispatch(ctx context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing command"), false
	}

	switch args[0] {
	case "profiles":
		return e.handleProfiles(ctx, args[1:])
	case "projects":
		return e.handleProjects(ctx, args[1:])
	case "auth":
		return e.handleAuth(ctx, args[1:])
	case "dialogs":
		return e.handleDialogs(ctx, args[1:])
	case "messages":
		return e.handleMessages(ctx, args[1:])
	case "me":
		return e.handleMe(ctx, args[1:])
	case "daemon":
		return e.handleDaemon(ctx, args[1:])
	case "audit":
		return e.handleAudit(ctx, args[1:])
	default:
		return e.errorResponse("", "InvalidInput", "unknown command"), false
	}
}

func (e *Executor) currentCwd() string {
	if e.cwd != "" {
		return e.cwd
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func (e *Executor) resolveEffectiveProfile(profileID string, explicit bool) (string, error) {
	trimmed := strings.TrimSpace(profileID)
	if explicit {
		return trimmed, nil
	}
	if trimmed != "" {
		return trimmed, nil
	}

	binding, ok, err := e.store.ResolveProjectBinding(e.currentCwd())
	if err != nil {
		return "", err
	}
	if !ok {
		return "qa-dev", nil
	}
	if _, err := e.store.Get(binding.ProfileID); err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			return binding.ProfileID, errProjectProfileMissing
		}
		return binding.ProfileID, err
	}
	return binding.ProfileID, nil
}

func (e *Executor) authorizedSession(profileID string) (tg.SessionRef, error) {
	view, err := e.store.Get(profileID)
	if err != nil {
		return tg.SessionRef{}, err
	}
	hasSession, err := e.store.SessionExists(profileID)
	if err != nil {
		return tg.SessionRef{}, err
	}
	if view.AuthorizationStatus != profile.AuthorizationAuthorized || !hasSession {
		return tg.SessionRef{}, errUnauthorizedProfile
	}

	return tg.SessionRef{
		ProfileID:   profileID,
		StorageRoot: view.StorageRoot,
		SessionPath: e.store.SessionPath(profileID),
	}, nil
}

func (e *Executor) requireTelegramConfig() (tg.RuntimeConfig, error) {
	apiIDRaw, ok := e.lookupEnv("MI_TELEGRAM_API_ID")
	if !ok || strings.TrimSpace(apiIDRaw) == "" {
		return tg.RuntimeConfig{}, errors.New("missing MI_TELEGRAM_API_ID")
	}

	apiHash, ok := e.lookupEnv("MI_TELEGRAM_API_HASH")
	if !ok || strings.TrimSpace(apiHash) == "" {
		return tg.RuntimeConfig{}, errors.New("missing MI_TELEGRAM_API_HASH")
	}

	apiID, err := strconv.Atoi(strings.TrimSpace(apiIDRaw))
	if err != nil {
		return tg.RuntimeConfig{}, errors.New("invalid MI_TELEGRAM_API_ID")
	}

	return tg.RuntimeConfig{
		APIID:   apiID,
		APIHash: apiHash,
	}, nil
}

func (e *Executor) resolvePeer(ctx context.Context, profileID string, runtimeConfig tg.RuntimeConfig, sessionRef tg.SessionRef, peerQuery string) (tg.Peer, output.Response, bool) {
	peer, err := e.telegram.ResolvePeer(ctx, runtimeConfig, sessionRef, tg.ResolvePeerRequest{
		Query: peerQuery,
	})
	if err == nil {
		return peer, output.Response{}, true
	}

	switch {
	case errors.Is(err, tg.ErrUnauthorized):
		return tg.Peer{}, e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized"), false
	case errors.Is(err, tg.ErrPeerNotFound):
		return tg.Peer{}, e.errorResponse(profileID, "PeerNotFound", "peer was not found"), false
	case errors.Is(err, tg.ErrPeerAmbiguous):
		return tg.Peer{}, e.errorResponse(profileID, "PeerAmbiguous", "peer query matched multiple dialogs"), false
	default:
		return tg.Peer{}, e.errorResponse(profileID, "TelegramListDialogsFailed", err.Error()), false
	}
}

func (e *Executor) isProtectedProfileForAutomation(profileID string) bool {
	return protectedAutomationProfiles[strings.TrimSpace(profileID)]
}

func (e *Executor) profileProtectedResponse(profileID string) output.Response {
	return e.errorResponse(profileID, "ProfileProtected", profileProtectedMessage)
}

func (e *Executor) withProfileLock(profileID string, jsonMode bool, queueTimeout time.Duration, fn func() output.Response) (output.Response, bool) {
	lock, queueMs, err := e.acquireProfileLock(profileID, queueTimeout)
	if err != nil {
		return e.mapStoreError(profileID, err), jsonMode
	}
	defer func() { _ = lock.Release() }()

	resp := fn()
	resp.QueueMs = queueMs.Milliseconds()
	return resp, jsonMode
}

func (e *Executor) acquireProfileLock(profileID string, queueTimeout time.Duration) (*profile.Lock, time.Duration, error) {
	mode := e.effectiveDaemonMode()
	if mode == "off" || e.daemonManager == nil {
		lock, err := e.store.AcquireLock(profileID)
		return lock, 0, err
	}

	if _, running, err := e.daemonManager.EnsureStarted(); err != nil || !running {
		if mode == "required" {
			if err == nil {
				err = daemon.ErrUnavailable
			}
			return nil, 0, err
		}
	}

	return e.store.AcquireQueuedLock(profileID, queueTimeout)
}

func (e *Executor) effectiveDaemonMode() string {
	if e.daemonMode != "" {
		return strings.ToLower(e.daemonMode)
	}
	if raw, ok := e.lookupEnv("MI_TELEGRAM_CLI_DAEMON"); ok && strings.TrimSpace(raw) != "" {
		return strings.ToLower(strings.TrimSpace(raw))
	}
	return "off"
}

func (e *Executor) defaultQueueTimeout() time.Duration {
	raw, ok := e.lookupEnv("MI_TELEGRAM_CLI_QUEUE_TIMEOUT_SECONDS")
	if !ok || strings.TrimSpace(raw) == "" {
		return 120 * time.Second
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || seconds < 0 {
		return 120 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func (e *Executor) promptValue(label string, current string) (string, error) {
	if strings.TrimSpace(current) != "" {
		return strings.TrimSpace(current), nil
	}
	if !e.interactive {
		return "", nil
	}
	return e.prompt(label)
}

func (e *Executor) mapTelegramUnauthorizedOr(profileID, code string, err error) output.Response {
	if errors.Is(err, tg.ErrUnauthorized) {
		return e.errorResponse(profileID, "UnauthorizedProfile", "profile is not authorized")
	}
	return e.errorResponse(profileID, code, err.Error())
}

func (e *Executor) mapStoreError(profileID string, err error) output.Response {
	switch {
	case errors.Is(err, profile.ErrProfileAlreadyExists):
		return e.errorResponse(profileID, "ProfileAlreadyExists", err.Error())
	case errors.Is(err, profile.ErrProfileNotFound):
		return e.errorResponse(profileID, "ProfileNotFound", err.Error())
	case errors.Is(err, profile.ErrProfileLocked):
		return e.errorResponse(profileID, "ProfileLocked", err.Error())
	case errors.Is(err, profile.ErrQueueTimeout):
		return e.errorResponse(profileID, "QueueTimeout", err.Error())
	case errors.Is(err, profile.ErrDaemonLeaseDenied):
		return e.errorResponse(profileID, "DaemonLeaseDenied", err.Error())
	case errors.Is(err, profile.ErrDaemonLeaseExpired):
		return e.errorResponse(profileID, "DaemonLeaseExpired", err.Error())
	case errors.Is(err, daemon.ErrUnavailable):
		return e.errorResponse(profileID, "DaemonUnavailable", err.Error())
	case errors.Is(err, errProjectProfileMissing):
		return e.errorResponse(profileID, "ProjectProfileMissing", "project binding references a missing profile")
	default:
		return e.errorResponse(profileID, "LocalStorageFailure", err.Error())
	}
}

func (e *Executor) errorResponse(profileID, code, message string) output.Response {
	return output.Response{
		OK:      false,
		Profile: profileID,
		Data:    nil,
		Error: &output.ResponseError{
			Code:    code,
			Message: message,
		},
	}
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func queueTimeoutFlag(fs *flag.FlagSet, defaultValue time.Duration) *int {
	return fs.Int("queue-timeout", int(defaultValue.Seconds()), "")
}

func durationFromSeconds(v int) time.Duration {
	if v < 0 {
		return -1
	}
	return time.Duration(v) * time.Second
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
