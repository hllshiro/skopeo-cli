package skopeo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/hllshiro/skopeo-cli/internal/console"
)

// DownloadOptions configures a single image download.
type DownloadOptions struct {
	SavePath       string
	Platform       string
	Overwrite      bool
	NoUploadScript bool
	Registry       string
}

// DownloadResult reports the outcome of a download.
type DownloadResult struct {
	Success     bool
	ArchiveFile string
	RepoPath    string
	ImageName   string
}

// FindSkopeo returns the path to the skopeo executable, or an error if it
// cannot be found in PATH.
func FindSkopeo() (string, error) {
	return exec.LookPath("skopeo")
}

// CopySkopeoToDir copies the skopeo executable into targetDir.
func CopySkopeoToDir(targetDir string) error {
	skopeoPath, err := FindSkopeo()
	if err != nil {
		return err
	}

	destName := "skopeo"
	if runtime.GOOS == "windows" {
		destName = "skopeo.exe"
	}
	destPath := filepath.Join(targetDir, destName)

	if fileExists(destPath) {
		return nil
	}
	return copyFile(skopeoPath, destPath)
}

// RepoPath strips a leading registry host from an image reference.
func RepoPath(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) > 1 {
		first := parts[0]
		if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
			return strings.Join(parts[1:], "/")
		}
	}
	return image
}

// ArchiveFileName returns the .tar filename derived from an image reference.
func ArchiveFileName(image string) string {
	r := RepoPath(image)
	r = strings.ReplaceAll(r, ":", "-")
	r = strings.ReplaceAll(r, "/", "_")
	return r + ".tar"
}

// BuildSkopeoArgs constructs the argument list for `skopeo copy`.
func BuildSkopeoArgs(platform, image, archiveFile string) []string {
	args := []string{"copy", "--all"}
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) >= 1 && parts[0] != "" {
			args = append(args, "--override-os", parts[0])
		}
		if len(parts) >= 2 && parts[1] != "" {
			args = append(args, "--override-arch", parts[1])
		}
	}
	args = append(args, "docker://"+image, "oci-archive:"+archiveFile)
	return args
}

// DownloadImage downloads a single image via skopeo.
func DownloadImage(image string, opts DownloadOptions) DownloadResult {
	repoPath := RepoPath(image)
	archiveFile := filepath.Join(opts.SavePath, ArchiveFileName(image))

	if fileExists(archiveFile) && !opts.Overwrite {
		console.ColorLog(fmt.Sprintf("File already exists, skipping: %s", archiveFile), "yellow")
		return DownloadResult{Success: true, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
	}

	args := BuildSkopeoArgs(opts.Platform, image, archiveFile)

	console.ColorLog(fmt.Sprintf("Downloading image: %s...", image), "green")

	cmd := exec.Command("skopeo", args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return DownloadResult{Success: false, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return DownloadResult{Success: false, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
	}

	if err := cmd.Start(); err != nil {
		console.ColorLog(fmt.Sprintf("skopeo not found or not in PATH: %v", err), "red")
		return DownloadResult{Success: false, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
	}

	var stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, stdoutPipe)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(io.MultiWriter(os.Stderr, &stderrBuf), stderrPipe)
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		console.ColorLog(fmt.Sprintf("Failed to download image: %s", image), "red")
		if stderrBuf.Len() > 0 {
			console.ColorLog(stderrBuf.String(), "red")
		}
		_ = os.Remove(archiveFile)
		return DownloadResult{Success: false, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
	}

	console.ColorLog(fmt.Sprintf("Downloaded: %s -> %s", image, archiveFile), "green")
	return DownloadResult{Success: true, ArchiveFile: archiveFile, RepoPath: repoPath, ImageName: image}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
