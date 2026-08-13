package upload

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hllshiro/skopeo-cli/internal/skopeo"
)

func makeResult(dir, fileName, repoPath string) skopeo.DownloadResult {
	return skopeo.DownloadResult{
		Success:     true,
		ArchiveFile: filepath.Join(dir, fileName),
		RepoPath:    repoPath,
		ImageName:   "docker.io/" + repoPath,
	}
}

func readScript(t *testing.T, dir string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "upload_all.ps1"))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func TestParseExistingUploadScriptMissing(t *testing.T) {
	got := ParseExistingUploadScript(filepath.Join(t.TempDir(), "nonexistent.ps1"))
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestParseExistingUploadScriptSingle(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "upload_all.ps1")
	content := "# Auto-generated upload script\n$images = @(\n    \"nginx-latest.tar|docker://docker.senjone.com/library/nginx\"\n);"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got := ParseExistingUploadScript(p)
	if !equalStrings(got, []string{"nginx-latest.tar|docker://docker.senjone.com/library/nginx"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseExistingUploadScriptMultiple(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "upload_all.ps1")
	content := "# Auto-generated upload script\n$images = @(\n    \"nginx-latest.tar|docker://docker.senjone.com/library/nginx\"\n    \"ubuntu-latest.tar|docker://docker.senjone.com/library/ubuntu\"\n    \"redis-alpine.tar|docker://docker.senjone.com/library/redis:alpine\"\n);"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got := ParseExistingUploadScript(p)
	want := []string{
		"nginx-latest.tar|docker://docker.senjone.com/library/nginx",
		"ubuntu-latest.tar|docker://docker.senjone.com/library/ubuntu",
		"redis-alpine.tar|docker://docker.senjone.com/library/redis:alpine",
	}
	if !equalStrings(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestParseExistingUploadScriptIgnoresOutside(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "upload_all.ps1")
	content := "# Auto-generated upload script\n$images = @(\n    \"nginx-latest.tar|docker://docker.senjone.com/library/nginx\"\n);\nfor ($i = 0; $i -lt $images.Count; $i++) {\n    $parts = $images[$i] -split '\\|'\n}"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got := ParseExistingUploadScript(p)
	if !equalStrings(got, []string{"nginx-latest.tar|docker://docker.senjone.com/library/nginx"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseExistingUploadScriptEmpty(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "upload_all.ps1")
	if err := os.WriteFile(p, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	got := ParseExistingUploadScript(p)
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestGenerateUploadScriptNew(t *testing.T) {
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "docker.senjone.com")

	content, ok := readScript(t, dir)
	if !ok {
		t.Fatal("script not created")
	}
	if !strings.Contains(content, "nginx-latest.tar") {
		t.Error("content missing nginx-latest.tar")
	}
	if !strings.Contains(content, "docker.senjone.com") {
		t.Error("content missing docker.senjone.com")
	}
}

func TestGenerateUploadScriptAppends(t *testing.T) {
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "docker.senjone.com")
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "ubuntu-latest.tar", "library/ubuntu")}, dir, "", "docker.senjone.com")

	content, _ := readScript(t, dir)
	if !strings.Contains(content, "nginx-latest.tar") {
		t.Error("content missing nginx-latest.tar")
	}
	if !strings.Contains(content, "ubuntu-latest.tar") {
		t.Error("content missing ubuntu-latest.tar")
	}
}

func TestGenerateUploadScriptNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "docker.senjone.com")
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "docker.senjone.com")

	content, _ := readScript(t, dir)
	if got := strings.Count(content, "nginx-latest.tar"); got != 1 {
		t.Errorf("nginx-latest.tar appears %d times, want 1", got)
	}
}

func TestGenerateUploadScriptPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "docker.senjone.com")
	GenerateUploadScript([]skopeo.DownloadResult{
		makeResult(dir, "ubuntu-latest.tar", "library/ubuntu"),
		makeResult(dir, "redis-alpine.tar", "library/redis:alpine"),
	}, dir, "", "docker.senjone.com")

	content, _ := readScript(t, dir)
	for _, want := range []string{"nginx-latest.tar", "ubuntu-latest.tar", "redis-alpine.tar", "$images.Count"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q", want)
		}
	}
}

func TestGenerateUploadScriptSkipsFailed(t *testing.T) {
	dir := t.TempDir()
	entries := []skopeo.DownloadResult{
		{Success: false, ArchiveFile: filepath.Join(dir, "failed.tar"), RepoPath: "library/failed", ImageName: "docker.io/library/failed"},
		makeResult(dir, "success.tar", "library/success"),
	}
	GenerateUploadScript(entries, dir, "", "docker.senjone.com")

	content, _ := readScript(t, dir)
	if strings.Contains(content, "failed.tar") {
		t.Error("content should not contain failed.tar")
	}
	if !strings.Contains(content, "success.tar") {
		t.Error("content missing success.tar")
	}
}

func TestGenerateUploadScriptAllFailed(t *testing.T) {
	dir := t.TempDir()
	entries := []skopeo.DownloadResult{
		{Success: false, ArchiveFile: filepath.Join(dir, "failed.tar"), RepoPath: "library/failed", ImageName: "docker.io/library/failed"},
	}
	GenerateUploadScript(entries, dir, "", "docker.senjone.com")

	if _, ok := readScript(t, dir); ok {
		t.Error("script should not exist")
	}
}

func TestGenerateUploadScriptRegistry(t *testing.T) {
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "", "custom.registry.io")

	content, _ := readScript(t, dir)
	if !strings.Contains(content, "custom.registry.io/library/nginx") {
		t.Error("content missing custom.registry.io/library/nginx")
	}
}

func TestGenerateUploadScriptCustomSkopeoPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("custom path only applies on non-Windows")
	}
	dir := t.TempDir()
	GenerateUploadScript([]skopeo.DownloadResult{makeResult(dir, "nginx-latest.tar", "library/nginx")}, dir, "/usr/local/bin/skopeo", "docker.senjone.com")

	content, _ := readScript(t, dir)
	if !strings.Contains(content, "/usr/local/bin/skopeo") {
		t.Error("content missing /usr/local/bin/skopeo")
	}
}

func TestUploadScriptSkopeoCmdWindows(t *testing.T) {
	if got := UploadScriptSkopeoCmd("", true); got != `"$PSScriptRoot\skopeo.exe"` {
		t.Errorf("got %q", got)
	}
	if got := UploadScriptSkopeoCmd(`D:\tools\skopeo.exe`, true); got != `"$PSScriptRoot\skopeo.exe"` {
		t.Errorf("got %q", got)
	}
}

func TestUploadScriptSkopeoCmdUnix(t *testing.T) {
	if got := UploadScriptSkopeoCmd("/usr/local/bin/skopeo", false); got != `"/usr/local/bin/skopeo"` {
		t.Errorf("got %q", got)
	}
	if got := UploadScriptSkopeoCmd("", false); got != "skopeo" {
		t.Errorf("got %q", got)
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
