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

func TestParseComposeFileIgnoresComments(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
# image: ghost:latest
services:
  web:
    image: nginx:latest   # image: also-ghost:latest
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileAnchors(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
x-base: &base
  image: nginx:latest

services:
  web1:
    <<: *base
  web2:
    image: redis:alpine
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest", "redis:alpine"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileAnchorOverridden(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
x-base: &base
  image: nginx:latest

services:
  web:
    <<: *base
    image: nginx:1.27
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:1.27"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileMultiMerge(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
x-a: &a
  image: nginx:latest
x-b: &b
  restart: always

services:
  web:
    <<: [*a, *b]
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileFoldedImage(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  web:
    image: >
      nginx:latest
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileEnvInterpolation(t *testing.T) {
	t.Setenv("SKOPEO_CLI_TEST_REGISTRY", "my.registry.io")
	t.Setenv("SKOPEO_CLI_TEST_TAG", "1.2.3")
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: ${SKOPEO_CLI_TEST_REGISTRY}/team/app:${SKOPEO_CLI_TEST_TAG}
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"my.registry.io/team/app:1.2.3"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileEnvBareVar(t *testing.T) {
	t.Setenv("SKOPEO_CLI_TEST_IMAGE", "redis:alpine")
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: $SKOPEO_CLI_TEST_IMAGE
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"redis:alpine"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileDefaultValues(t *testing.T) {
	os.Unsetenv("SKOPEO_CLI_TEST_UNSET_1")
	os.Unsetenv("SKOPEO_CLI_TEST_UNSET_2")
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  a:
    image: nginx:${SKOPEO_CLI_TEST_UNSET_1:-latest}
  b:
    image: redis:${SKOPEO_CLI_TEST_UNSET_2-alpine}
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest", "redis:alpine"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileDefaultEmptyIsUnset(t *testing.T) {
	// ${VAR:-default} uses the default when the variable is empty,
	// ${VAR-default} keeps the empty value.
	t.Setenv("SKOPEO_CLI_TEST_EMPTY", "")
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  a:
    image: nginx:${SKOPEO_CLI_TEST_EMPTY:-latest}
  b:
    image: redis:${SKOPEO_CLI_TEST_EMPTY-alpine}
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:latest", "redis:"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileDotEnv(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, ".env", "SKOPEO_CLI_TEST_DOTENV_IMAGE=ubuntu:22.04\n")
	writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: ${SKOPEO_CLI_TEST_DOTENV_IMAGE}
`)
	os.Unsetenv("SKOPEO_CLI_TEST_DOTENV_IMAGE")
	got, _ := ParseComposeFile(filepath.Join(dir, "docker-compose.yml"), "")
	if !equalStrings(got, []string{"ubuntu:22.04"}) {
		t.Errorf("got %v", got)
	}
}

func TestParseComposeFileDollarEscape(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", `
services:
  app:
    image: "nginx:$${SKOPEO_CLI_TEST_UNUSED:-latest}"
`)
	got, _ := ParseComposeFile(p, "")
	if !equalStrings(got, []string{"nginx:${SKOPEO_CLI_TEST_UNUSED:-latest}"}) {
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

func TestParseComposeFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeCompose(t, dir, "docker-compose.yml", "services:\n  web: [unclosed\n")
	if _, err := ParseComposeFile(p, ""); err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseComposeFileNotFound(t *testing.T) {
	_, err := ParseComposeFile(filepath.Join(t.TempDir(), "nonexistent.yml"), "")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestResolveComposeFileDirectory(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "compose.yaml", "services: {}\n")
	got, err := ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "compose.yaml" {
		t.Errorf("got %q, want compose.yaml", got)
	}
}

func TestResolveComposeFileDirectoryFallback(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", "services: {}\n")
	got, err := ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "docker-compose.yml" {
		t.Errorf("got %q, want docker-compose.yml", got)
	}
}

func TestResolveComposeFileNone(t *testing.T) {
	if _, err := ResolveComposeFile(t.TempDir()); err == nil {
		t.Error("expected error for directory without compose file")
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
