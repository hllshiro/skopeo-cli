//go:build integration

package skopeo

import "testing"

func TestDownloadImageActual(t *testing.T) {
	dir := t.TempDir()
	result := DownloadImage("docker.io/library/hello-world:latest", DownloadOptions{
		SavePath:  dir,
		Overwrite: true,
	})
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.RepoPath != "library/hello-world:latest" {
		t.Errorf("RepoPath = %q, want %q", result.RepoPath, "library/hello-world:latest")
	}
}

func TestDownloadImageInvalidImage(t *testing.T) {
	dir := t.TempDir()
	result := DownloadImage("docker.io/library/nonexistent-image-xyz-123:latest", DownloadOptions{
		SavePath:  dir,
		Overwrite: true,
	})
	if result.Success {
		t.Error("expected failure for invalid image")
	}
}
