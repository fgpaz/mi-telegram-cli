package daemon

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("daemon unavailable")

type State struct {
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Token        string    `json:"token"`
	PID          int       `json:"pid"`
	StartedAtUTC time.Time `json:"startedAtUtc"`
}

type Manager struct {
	baseRoot string
	now      func() time.Time
}

func NewManager(baseRoot string, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now().UTC
	}
	return &Manager{baseRoot: baseRoot, now: now}
}

func (m *Manager) StatePath() string {
	return filepath.Join(m.baseRoot, "daemon", "state.json")
}

func (m *Manager) Status() (State, bool, error) {
	state, err := m.readState()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, false, nil
		}
		return State{}, false, err
	}
	if err := ping(state); err != nil {
		return state, false, nil
	}
	return state, true, nil
}

func (m *Manager) Stop() error {
	state, err := m.readState()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := send(state, "STOP"); err != nil {
		_ = os.Remove(m.StatePath())
		return nil
	}
	_ = os.Remove(m.StatePath())
	return nil
}

func (m *Manager) Serve() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	state := State{
		Host:         "127.0.0.1",
		Port:         addr.Port,
		Token:        fmt.Sprintf("%d-%d", m.now().UnixNano(), os.Getpid()),
		PID:          os.Getpid(),
		StartedAtUTC: m.now(),
	}
	if err := m.writeState(state); err != nil {
		return err
	}
	defer os.Remove(m.StatePath())

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		stop := handleConn(conn, state.Token)
		if stop {
			return nil
		}
	}
}

func (m *Manager) EnsureStarted() (State, bool, error) {
	state, running, err := m.Status()
	if err != nil || running {
		return state, running, err
	}
	exe, err := os.Executable()
	if err != nil {
		return State{}, false, err
	}
	proc, err := os.StartProcess(exe, []string{exe, "daemon", "run-internal"}, &os.ProcAttr{
		Files: []*os.File{nil, nil, nil},
		Env:   os.Environ(),
	})
	if err != nil {
		return State{}, false, err
	}
	_ = proc.Release()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state, running, err = m.Status()
		if err != nil {
			return State{}, false, err
		}
		if running {
			return state, true, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return State{}, false, ErrUnavailable
}

func (m *Manager) readState() (State, error) {
	raw, err := os.ReadFile(m.StatePath())
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (m *Manager) writeState(state State) error {
	if err := os.MkdirAll(filepath.Dir(m.StatePath()), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.StatePath(), raw, 0o600)
}

func ping(state State) error {
	return send(state, "PING")
}

func send(state State, command string) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(state.Host, strconv.Itoa(state.Port)), 500*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "%s %s\n", state.Token, command); err != nil {
		return err
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "OK" {
		return ErrUnavailable
	}
	return nil
}

func handleConn(conn net.Conn, token string) bool {
	defer conn.Close()
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return false
	}
	parts := strings.Fields(line)
	if len(parts) != 2 || parts[0] != token {
		_, _ = fmt.Fprintln(conn, "ERR")
		return false
	}
	_, _ = fmt.Fprintln(conn, "OK")
	return parts[1] == "STOP"
}
