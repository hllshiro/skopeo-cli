package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCompose(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseComposeFileSimple(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  nginx:
    image: nginx:latest
  redis:
    image: redis:alpine
`)
	got, err := ParseComposeFile(p, "")
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, []string{"nginx:latest", "redis:alpine"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileDeduplicates(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  web1:
    image: nginx:latest
  web2:
    image: nginx:latest
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileStripsDoubleQuotes(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: "nginx:latest"
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileStripsSingleQuotes(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: 'nginx:latest'
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileFilter(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  nginx:
    image: nginx:latest
  redis:
    image: redis:alpine
  postgres:
    image: postgres:15
`)
	got, _ := ParseComposeFile(p, "nginx")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileFilterCaseSensitive(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  nginx:
    image: nginx:latest
`)
	got, _ := ParseComposeFile(p, "Nginx")
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestParseComposeFileEmpty(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", "services: {}")
	got, _ := ParseComposeFile(p, "")
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestParseComposeFileRegistryPrefix(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: docker.io/library/nginx:latest
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"docker.io/library/nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileNotFound(t *testing.T) {
	_, err := ParseComposeFile(filepath.Join(t.TempDir(), "nonexistent.yml"), "")
	if err == nil {
		t.Error("expected error for non-existent file")
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
