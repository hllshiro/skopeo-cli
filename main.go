package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hllshiro/skopeo-cli/internal/cli"
	"github.com/hllshiro/skopeo-cli/internal/compose"
	"github.com/hllshiro/skopeo-cli/internal/console"
	"github.com/hllshiro/skopeo-cli/internal/skopeo"
	"github.com/hllshiro/skopeo-cli/internal/upload"
)

const (
	version         = "2.0.0"
	defaultRegistry = "docker.senjone.com"
)

type options struct {
	savePath       string
	platform       string
	filter         string
	overwrite      bool
	noUploadScript bool
	registry       string
}

func main() {
	parsed := cli.ParseArgs(os.Args[1:])

	if parsed.Command == "version" || cli.HasFlag(parsed.Options, "version") {
		fmt.Println(version)
		return
	}
	if parsed.Command == "help" || cli.HasFlag(parsed.Options, "help") {
		printUsage()
		return
	}

	var err error
	switch parsed.Command {
	case "download":
		err = downloadCommand(parsed.Positional, parseOptions(parsed.Options))
	case "compose":
		err = composeCommand(parsed.Positional, parseOptions(parsed.Options))
	default:
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		os.Exit(1)
	}
}

func parseOptions(o map[string]string) options {
	return options{
		savePath:       cli.OptionString(o, "save"),
		platform:       cli.OptionString(o, "platform"),
		filter:         cli.OptionString(o, "filter"),
		overwrite:      cli.HasFlag(o, "overwrite"),
		noUploadScript: cli.HasFlag(o, "no-upload-script"),
		registry:       cli.OptionStringDefault(o, "registry", defaultRegistry),
	}
}

func printUsage() {
	console.ColorLog("Usage: skopeo-cli <download|compose> [options]", "yellow")
	console.ColorLog("  download <image>    Download a single Docker image", "white")
	console.ColorLog("  compose <file>      Batch download from compose file", "white")
	console.ColorLog("  --registry <host>   Target registry (default: docker.senjone.com)", "white")
}

var errUsage = errors.New("usage error")

func downloadCommand(positional []string, opts options) error {
	if len(positional) == 0 {
		console.ColorLog("Usage: skopeo-cli download <image>", "yellow")
		return errUsage
	}
	image := positional[0]

	savePath := opts.savePath
	if savePath == "" {
		savePath = cli.DefaultDownloadPath()
	}
	outputDir := filepath.Join(savePath, "docker-image-will-upload")
	ensureDir(outputDir)

	skopeoPath, err := requireSkopeo(outputDir)
	if err != nil {
		return err
	}

	result := skopeo.DownloadImage(image, skopeo.DownloadOptions{
		SavePath:       outputDir,
		Platform:       opts.platform,
		Overwrite:      opts.overwrite,
		NoUploadScript: opts.noUploadScript,
		Registry:       opts.registry,
	})

	if result.Success {
		console.ColorLog(fmt.Sprintf("Recorded: %s -> %s/%s", image, opts.registry, result.RepoPath), "green")
		if !opts.noUploadScript {
			upload.GenerateUploadScript([]skopeo.DownloadResult{result}, outputDir, scriptSkopeoPath(skopeoPath), opts.registry)
		}
	}

	console.ColorLog(fmt.Sprintf("Output directory: %s", outputDir), "cyan")
	return nil
}

func composeCommand(positional []string, opts options) error {
	if len(positional) == 0 {
		console.ColorLog("Usage: skopeo-cli compose <file>", "yellow")
		return errUsage
	}
	file := positional[0]

	console.ColorLog(fmt.Sprintf("Parsing %s ...", file), "cyan")
	if opts.filter != "" {
		console.ColorLog(fmt.Sprintf("Applying filter: *%s*", opts.filter), "magenta")
	}

	images, err := compose.ParseComposeFile(file, opts.filter)
	if err != nil {
		console.ColorLog(err.Error(), "red")
		return err
	}
	if len(images) == 0 {
		console.ColorLog("No images found.", "yellow")
		return nil
	}

	savePath := opts.savePath
	if savePath == "" {
		savePath = cli.DefaultDownloadPath()
	}
	outputDir := filepath.Join(savePath, "docker-image-will-upload")
	ensureDir(outputDir)

	skopeoPath, err := requireSkopeo(outputDir)
	if err != nil {
		return err
	}

	console.ColorLog(fmt.Sprintf("Found %d matching images, starting download...", len(images)), "cyan")
	fmt.Println("==================================================")

	results := make([]skopeo.DownloadResult, 0, len(images))
	for i, img := range images {
		console.ColorLog(fmt.Sprintf("[%d/%d] Processing: %s", i+1, len(images), img), "cyan")
		result := skopeo.DownloadImage(img, skopeo.DownloadOptions{
			SavePath:       outputDir,
			Platform:       opts.platform,
			Overwrite:      opts.overwrite,
			NoUploadScript: opts.noUploadScript,
			Registry:       opts.registry,
		})
		results = append(results, result)
		if !result.Success {
			console.ColorLog(fmt.Sprintf("Warning: %s download failed.", img), "yellow")
		}
		fmt.Println("--------------------------------------------------")
	}

	succeeded := 0
	for _, r := range results {
		if r.Success {
			succeeded++
		}
	}
	failed := len(results) - succeeded
	if failed > 0 {
		console.ColorLog(fmt.Sprintf("Done! Success: %d, Failed: %d", succeeded, failed), "yellow")
	} else {
		console.ColorLog("All downloads completed!", "green")
	}

	if !opts.noUploadScript {
		upload.GenerateUploadScript(results, outputDir, scriptSkopeoPath(skopeoPath), opts.registry)
	}

	console.ColorLog(fmt.Sprintf("Output directory: %s", outputDir), "cyan")
	return nil
}

// scriptSkopeoPath returns the skopeo path to embed in the upload script. On
// non-Windows the script relies on skopeo being in PATH, so it is empty.
func scriptSkopeoPath(skopeoPath string) string {
	if runtime.GOOS == "windows" {
		return skopeoPath
	}
	return ""
}

func requireSkopeo(outputDir string) (string, error) {
	skopeoPath, err := skopeo.FindSkopeo()
	if err != nil {
		console.ColorLog("Error: skopeo not found. Please install skopeo first.", "red")
		console.ColorLog("  Windows: https://github.com/containers/skopeo/blob/main/install.md", "white")
		console.ColorLog("  Linux:   sudo apt install skopeo / sudo yum install skopeo", "white")
		return "", err
	}
	console.ColorLog(fmt.Sprintf("Found skopeo: %s", skopeoPath), "cyan")

	if runtime.GOOS == "windows" {
		console.ColorLog("Copying skopeo to output directory...", "cyan")
		if err := skopeo.CopySkopeoToDir(outputDir); err != nil {
			return "", err
		}
	}
	return skopeoPath, nil
}

func ensureDir(path string) {
	_ = os.MkdirAll(path, 0755)
}
