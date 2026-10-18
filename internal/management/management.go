package management

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"regexp"
	"runtime"
	"time"
)

var (
	buildVersion = "0.1.0"
	buildCommit  = "unknown"
	buildTime    = "unknown"
	vcsModified  = "unknown"
)

//go:embed provider-manifest.json
var providerManifest []byte

type versionData struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildTime   string `json:"build_time"`
	Go          string `json:"go"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	VCSModified string `json:"vcs_modified"`
}

type envelope struct {
	OK   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Execute handles local, read-only management commands before runtime setup.
// handled is false when args do not select a management command.
func Execute(args []string, stdout io.Writer) (handled bool, exitCode int) {
	if len(args) == 0 {
		return false, 0
	}
	command := args[0]
	switch command {
	case "version", "--version":
		if len(args) != 1 {
			return true, writeError(stdout, "parametro_invalido", "version accepts no arguments")
		}
		return true, writeJSON(stdout, envelope{OK: true, Data: currentVersion()})
	case "provider-manifest":
		if len(args) != 1 {
			return true, writeError(stdout, "parametro_invalido", "provider-manifest accepts no arguments")
		}
		return true, writeManifest(stdout)
	case "doctor":
		if len(args) != 1 {
			return true, writeError(stdout, "parametro_invalido", "doctor accepts no arguments")
		}
		return true, writeJSON(stdout, envelope{OK: true, Data: doctor()})
	default:
		return false, 0
	}
}

func currentVersion() versionData {
	return versionData{
		Version:     buildVersion,
		Commit:      buildCommit,
		BuildTime:   buildTime,
		Go:          runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		VCSModified: vcsModified,
	}
}

func doctor() map[string]any {
	version := currentVersion()
	complete := version.Version != "" && commitPattern.MatchString(version.Commit) && version.VCSModified == "false"
	if _, err := time.Parse(time.RFC3339, version.BuildTime); err != nil {
		complete = false
	}
	status := "ok"
	gaps := []string{}
	checkStatus := "ok"
	if !complete {
		status = "explicit_incomplete"
		checkStatus = "explicit_incomplete"
		gaps = append(gaps, "provenance_unknown")
	}
	return map[string]any{
		"status":    status,
		"read_only": true,
		"redacted":  true,
		"checks": []map[string]string{{
			"name":   "build_provenance",
			"status": checkStatus,
		}},
		"gaps": gaps,
	}
}

func writeManifest(w io.Writer) int {
	var manifest map[string]any
	if err := json.Unmarshal(providerManifest, &manifest); err != nil {
		return writeError(w, "error_interno", "provider manifest is unavailable")
	}
	manifest["provider_version"] = buildVersion
	return writeJSON(w, manifest)
}

func writeJSON(w io.Writer, value any) int {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return writeError(w, "error_interno", "management output is unavailable")
	}
	if _, err := w.Write(buffer.Bytes()); err != nil {
		return 1
	}
	return 0
}

func writeError(w io.Writer, reason, message string) int {
	payload := map[string]any{
		"ok":          false,
		"reason_code": reason,
		"detail":      map[string]string{"message": message},
	}
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return 1
	}
	if _, err := w.Write(buffer.Bytes()); err != nil {
		return 1
	}
	return 2
}
