package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/hllshiro/skopeo-cli/internal/console"
	"github.com/hllshiro/skopeo-cli/internal/skopeo"
)

var uploadEntryRe = regexp.MustCompile(`^\s*"([^"]+?\|docker://[^"]+)"$`)

// ParseExistingUploadScript extracts existing entries from an upload script.
func ParseExistingUploadScript(scriptPath string) []string {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil
	}

	entries := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		m := uploadEntryRe.FindStringSubmatch(line)
		if len(m) > 1 {
			entries = append(entries, m[1])
		}
	}
	return entries
}

// UploadScriptSkopeoCmd returns the skopeo invocation used in the upload
// script.
func UploadScriptSkopeoCmd(skopeoPath string, isWin bool) string {
	if isWin {
		return `"$PSScriptRoot\skopeo.exe"`
	}
	if skopeoPath != "" {
		return `"` + skopeoPath + `"`
	}
	return "skopeo"
}

// GenerateUploadScript writes (or merges into) an upload_all.ps1 script.
func GenerateUploadScript(entries []skopeo.DownloadResult, targetDir, skopeoPath, registry string) {
	successful := []skopeo.DownloadResult{}
	for _, e := range entries {
		if e.Success {
			successful = append(successful, e)
		}
	}
	if len(successful) == 0 {
		return
	}

	scriptPath := filepath.Join(targetDir, "upload_all.ps1")

	existing := ParseExistingUploadScript(scriptPath)

	newEntries := []string{}
	for _, e := range successful {
		fileName := filepath.Base(e.ArchiveFile)
		newEntries = append(newEntries, fileName+"|docker://"+registry+"/"+e.RepoPath)
	}

	allEntries := append([]string{}, existing...)
	for _, ne := range newEntries {
		if !contains(allEntries, ne) {
			allEntries = append(allEntries, ne)
		}
	}

	skopeoCmd := UploadScriptSkopeoCmd(skopeoPath, runtime.GOOS == "windows")

	lines := []string{
		"# Auto-generated upload script by skopeo-cli",
		"# Place this script in the same directory as the image files",
		"",
		"Set-Location $PSScriptRoot",
		"",
		"$images = @(",
	}
	for _, e := range allEntries {
		lines = append(lines, "    \""+e+"\"")
	}
	lines = append(lines,
		");",
		"",
		"for ($i = 0; $i -lt $images.Count; $i++) {",
		`    $parts = $images[$i] -split '\|'`,
		"    Write-Host \"[$($i+1)/$($images.Count)] Uploading: $($parts[0]) ...\" -ForegroundColor Cyan",
		"    & "+skopeoCmd+` copy --all "oci-archive:$($parts[0])" $parts[1]`,
		"    if ($LASTEXITCODE -ne 0) {",
		"        Write-Host \"Upload failed: $($parts[0])\" -ForegroundColor Red",
		"    }",
		"}",
		"Write-Host \"All uploads completed!\" -ForegroundColor Green",
	)

	_ = os.WriteFile(scriptPath, []byte(strings.Join(lines, "\n")), 0644)
	console.ColorLog(fmt.Sprintf("Generated upload script: %s", scriptPath), "green")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
