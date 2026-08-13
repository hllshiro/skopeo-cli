package cli

import (
	"strings"
	"testing"
)

func TestParseArgsSimpleCommand(t *testing.T) {
	r := ParseArgs([]string{"download"})
	if r.Command != "download" {
		t.Errorf("Command = %q, want %q", r.Command, "download")
	}
	if len(r.Positional) != 0 {
		t.Errorf("Positional = %v, want empty", r.Positional)
	}
	if len(r.Options) != 0 {
		t.Errorf("Options = %v, want empty", r.Options)
	}
}

func TestParseArgsCommandWithPositional(t *testing.T) {
	r := ParseArgs([]string{"download", "nginx:latest"})
	if r.Command != "download" {
		t.Errorf("Command = %q, want %q", r.Command, "download")
	}
	if !equalStrings(r.Positional, []string{"nginx:latest"}) {
		t.Errorf("Positional = %v, want [nginx:latest]", r.Positional)
	}
	if len(r.Options) != 0 {
		t.Errorf("Options = %v, want empty", r.Options)
	}
}

func TestParseArgsMultiplePositional(t *testing.T) {
	r := ParseArgs([]string{"compose", "docker-compose.yml", "extra"})
	if r.Command != "compose" {
		t.Errorf("Command = %q, want %q", r.Command, "compose")
	}
	if !equalStrings(r.Positional, []string{"docker-compose.yml", "extra"}) {
		t.Errorf("Positional = %v", r.Positional)
	}
}

func TestParseArgsKeyValue(t *testing.T) {
	r := ParseArgs([]string{"download", "nginx", "--save", "/tmp/images"})
	if r.Command != "download" {
		t.Errorf("Command = %q, want %q", r.Command, "download")
	}
	if !equalStrings(r.Positional, []string{"nginx"}) {
		t.Errorf("Positional = %v, want [nginx]", r.Positional)
	}
	if r.Options["save"] != "/tmp/images" {
		t.Errorf("Options[save] = %q, want %q", r.Options["save"], "/tmp/images")
	}
}

func TestParseArgsBooleanFlag(t *testing.T) {
	r := ParseArgs([]string{"download", "nginx", "--overwrite"})
	if r.Command != "download" {
		t.Errorf("Command = %q, want %q", r.Command, "download")
	}
	if r.Options["overwrite"] != "true" {
		t.Errorf("Options[overwrite] = %q, want %q", r.Options["overwrite"], "true")
	}
}

func TestParseArgsMixed(t *testing.T) {
	r := ParseArgs([]string{
		"compose", "docker-compose.yml",
		"--save", "/tmp",
		"--overwrite",
		"--filter", "nginx",
		"--registry", "my.registry.com",
	})
	if r.Command != "compose" {
		t.Errorf("Command = %q, want %q", r.Command, "compose")
	}
	if !equalStrings(r.Positional, []string{"docker-compose.yml"}) {
		t.Errorf("Positional = %v", r.Positional)
	}
	if r.Options["save"] != "/tmp" {
		t.Errorf("Options[save] = %q", r.Options["save"])
	}
	if r.Options["overwrite"] != "true" {
		t.Errorf("Options[overwrite] = %q", r.Options["overwrite"])
	}
	if r.Options["filter"] != "nginx" {
		t.Errorf("Options[filter] = %q", r.Options["filter"])
	}
	if r.Options["registry"] != "my.registry.com" {
		t.Errorf("Options[registry] = %q", r.Options["registry"])
	}
}

func TestParseArgsEmpty(t *testing.T) {
	r := ParseArgs([]string{})
	if r.Command != "" {
		t.Errorf("Command = %q, want empty", r.Command)
	}
	if len(r.Positional) != 0 {
		t.Errorf("Positional = %v, want empty", r.Positional)
	}
	if len(r.Options) != 0 {
		t.Errorf("Options = %v, want empty", r.Options)
	}
}

func TestParseArgsFlagsBeforeCommand(t *testing.T) {
	r := ParseArgs([]string{"--save", "/tmp", "download", "nginx"})
	if r.Command != "download" {
		t.Errorf("Command = %q, want %q", r.Command, "download")
	}
	if !equalStrings(r.Positional, []string{"nginx"}) {
		t.Errorf("Positional = %v, want [nginx]", r.Positional)
	}
	if r.Options["save"] != "/tmp" {
		t.Errorf("Options[save] = %q, want %q", r.Options["save"], "/tmp")
	}
}

func TestParseArgsMultipleBooleanFlags(t *testing.T) {
	r := ParseArgs([]string{"download", "nginx", "--overwrite", "--no-upload-script"})
	if r.Options["overwrite"] != "true" {
		t.Errorf("Options[overwrite] = %q", r.Options["overwrite"])
	}
	if r.Options["no-upload-script"] != "true" {
		t.Errorf("Options[no-upload-script] = %q", r.Options["no-upload-script"])
	}
}

func TestParseArgsKeyEqualsValue(t *testing.T) {
	r := ParseArgs([]string{"download", "nginx", "--save=/tmp/images", "--registry=my.reg.io"})
	if r.Options["save"] != "/tmp/images" {
		t.Errorf("Options[save] = %q", r.Options["save"])
	}
	if r.Options["registry"] != "my.reg.io" {
		t.Errorf("Options[registry] = %q", r.Options["registry"])
	}
}

func TestHasFlag(t *testing.T) {
	opts := map[string]string{"overwrite": "true", "no-upload-script": "true"}
	if !HasFlag(opts, "overwrite") {
		t.Error("HasFlag(overwrite) = false, want true")
	}
	if HasFlag(opts, "missing") {
		t.Error("HasFlag(missing) = true, want false")
	}
}

func TestHasFlagExplicitFalse(t *testing.T) {
	opts := map[string]string{"overwrite": "false"}
	if HasFlag(opts, "overwrite") {
		t.Error("HasFlag(overwrite=false) = true, want false")
	}
}

func TestDefaultDownloadPathUnix(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("USERPROFILE", "")
	if got := defaultDownloadPath("linux"); got != "/home/testuser/Downloads" {
		t.Errorf("got %q, want %q", got, "/home/testuser/Downloads")
	}
}

func TestDefaultDownloadPathWindows(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\testuser`)
	if got := defaultDownloadPath("windows"); got != `C:\Users\testuser\Downloads` {
		t.Errorf("got %q, want %q", got, `C:\Users\testuser\Downloads`)
	}
}

func TestDefaultDownloadPathWindowsFallbackHome(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOME", "/home/testuser")
	if got := defaultDownloadPath("windows"); got != "/home/testuser/Downloads" {
		t.Errorf("got %q, want %q", got, "/home/testuser/Downloads")
	}
}

func TestDefaultDownloadPathNoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	got := defaultDownloadPath("linux")
	if !strings.HasSuffix(got, "/Downloads") {
		t.Errorf("got %q, want suffix /Downloads", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
