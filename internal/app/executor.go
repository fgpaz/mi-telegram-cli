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
}

var errUnauthorizedProfile = errors.New("unauthorized profile")

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

	return &Executor{
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
	}
}

func (e *Executor) Execute(ctx context.Context, args []string) int {
	resp, jsonMode := e.dispatch(ctx, args)
	if jsonMode {
		_ = output.WriteJSON(e.stdout, resp)
	} else {
		_ = output.WriteHuman(e.stdout, resp)
	}

	if resp.OK {
		return 0
	}
	return 1
}

func (e *Executor) dispatch(ctx context.Context, args []string) (output.Response, bool) {
	if len(args) == 0 {
		return e.errorResponse("", "InvalidInput", "missing command"), false
	}

	switch args[0] {
	case "profiles":
		return e.handleProfiles(ctx, args[1:])
	case "auth":
		return e.handleAuth(ctx, args[1:])
	case "dialogs":
		return e.handleDialogs(ctx, args[1:])
	case "messages":
		return e.handleMessages(ctx, args[1:])
	case "me":
		return e.handleMe(ctx, args[1:])
	default:
		return e.errorResponse("", "InvalidInput", "unknown command"), false
	}
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

func (e *Executor) withProfileLock(profileID string, jsonMode bool, fn func() output.Response) (output.Response, bool) {
	lock, err := e.store.AcquireLock(profileID)
	if err != nil {
		return e.mapStoreError(profileID, err), jsonMode
	}
	defer func() { _ = lock.Release() }()

	return fn(), jsonMode
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

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
