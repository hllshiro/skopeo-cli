package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hllshiro/skopeo-cli/internal/console"
	"github.com/hllshiro/skopeo-cli/internal/skopeo"
)

// uploadEntryRe matches the quoted "<archive>|docker://<dest>" lines embedded
// in both upload_all.ps1 and upload_all.sh. The same line format is used in
// both scripts so existing entries can be merged from either one.
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

// GenerateUploadScript writes (or merges into) both upload_all.ps1 and
// upload_all.sh scripts in targetDir. Both scripts share the same entries and
// prefer a bundled skopeo binary (skopeo.exe / skopeo) next to the script
// before falling back to skopeo on PATH.
func GenerateUploadScript(entries []skopeo.DownloadResult, targetDir, registry string) {
	successful := []skopeo.DownloadResult{}
	for _, e := range entries {
		if e.Success {
			successful = append(successful, e)
		}
	}
	if len(successful) == 0 {
		return
	}

	existing := []string{}
	for _, name := range []string{"upload_all.ps1", "upload_all.sh"} {
		existing = append(existing, ParseExistingUploadScript(filepath.Join(targetDir, name))...)
	}

	newEntries := []string{}
	for _, e := range successful {
		fileName := filepath.Base(e.ArchiveFile)
		newEntries = append(newEntries, fileName+"|docker://"+registry+"/"+e.RepoPath)
	}

	allEntries := []string{}
	seen := map[string]bool{}
	for _, e := range append(existing, newEntries...) {
		if !seen[e] {
			seen[e] = true
			allEntries = append(allEntries, e)
		}
	}

	ps1Path := filepath.Join(targetDir, "upload_all.ps1")
	_ = os.WriteFile(ps1Path, []byte(renderPowerShell(allEntries)), 0644)
	console.ColorLog(fmt.Sprintf("Generated upload script: %s", ps1Path), "green")

	shPath := filepath.Join(targetDir, "upload_all.sh")
	_ = os.WriteFile(shPath, []byte(renderShell(allEntries)), 0755)
	console.ColorLog(fmt.Sprintf("Generated upload script: %s", shPath), "green")
}

// renderPowerShell builds the upload_all.ps1 script body.
func renderPowerShell(entries []string) string {
	lines := []string{
		"# Auto-generated upload script by skopeo-cli",
		"# Place this script in the same directory as the image files",
		"",
		"Set-Location $PSScriptRoot",
		"",
		`$skopeo = "skopeo"`,
		`if (Test-Path "$PSScriptRoot\skopeo.exe") {`,
		`    $skopeo = "$PSScriptRoot\skopeo.exe"`,
		"}",
		"",
		"$images = @(",
	}
	for _, e := range entries {
		lines = append(lines, "    \""+e+"\"")
	}
	lines = append(lines,
		");",
		"",
		"for ($i = 0; $i -lt $images.Count; $i++) {",
		`    $parts = $images[$i] -split '\|'`,
		"    Write-Host \"[$($i+1)/$($images.Count)] Uploading: $($parts[0]) ...\" -ForegroundColor Cyan",
		`    & $skopeo copy --all "oci-archive:$($parts[0])" $parts[1]`,
		"    if ($LASTEXITCODE -ne 0) {",
		"        Write-Host \"Upload failed: $($parts[0])\" -ForegroundColor Red",
		"    }",
		"}",
		"Write-Host \"All uploads completed!\" -ForegroundColor Green",
	)
	return strings.Join(lines, "\n")
}

// renderShell builds the POSIX-sh upload_all.sh script body.
func renderShell(entries []string) string {
	lines := []string{
		"#!/usr/bin/env sh",
		"# Auto-generated upload script by skopeo-cli",
		"# Place this script in the same directory as the image files",
		"",
		"set -u",
		"",
		`SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)`,
		"",
		"SKOPEO=skopeo",
		`if [ -x "$SCRIPT_DIR/skopeo" ]; then`,
		`    SKOPEO="$SCRIPT_DIR/skopeo"`,
		"fi",
		"",
		"images=$(cat <<'EOF'",
	}
	for _, e := range entries {
		lines = append(lines, "\""+e+"\"")
	}
	lines = append(lines,
		"EOF",
		")",
		"",
		"i=0",
		"total=0",
		"for _ in $images; do",
		"    total=$((total + 1))",
		"done",
		"",
		"for entry in $images; do",
		"    i=$((i + 1))",
		"    archive=${entry%%|*}",
		"    archive=${archive#?}",
		"    dest=${entry#*|}",
		"    dest=${dest%?}",
		`    printf '[%d/%d] Uploading: %s ...\n' "$i" "$total" "$archive"`,
		`    "$SKOPEO" copy --all "oci-archive:$archive" "$dest"`,
		"    if [ $? -ne 0 ]; then",
		`        printf 'Upload failed: %s\n' "$archive"`,
		"    fi",
		"done",
		"",
		`printf 'All uploads completed!\n'`,
	)
	return strings.Join(lines, "\n")
}
