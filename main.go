package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hllshiro/skopeo-cli/internal/cli"
	"github.com/hllshiro/skopeo-cli/internal/compose"
	"github.com/hllshiro/skopeo-cli/internal/console"
	"github.com/hllshiro/skopeo-cli/internal/skopeo"
	"github.com/hllshiro/skopeo-cli/internal/upload"
)

const (
	version         = "2.2.0"
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
		switch parsed.Command {
		case "download":
			printDownloadUsage()
		case "compose":
			printComposeUsage()
		default:
			printUsage()
		}
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
		if err != errUsage {
			console.ColorLogErr("Error: "+err.Error(), "red")
		}
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

const usageText = `skopeo-cli - download Docker images and generate offline upload scripts

Usage:
  skopeo-cli <command> [options]

Commands:
  download <image>    Download a single Docker image
  compose <file>      Batch download images from a compose file

Options:
  --save <path>           Save directory (default: ~/Downloads)
  --platform <os/arch>    Target platform (default: all)
  --registry <host>       Target registry (default: docker.senjone.com)
  --overwrite             Overwrite existing files
  --no-upload-script      Skip generating the upload script

compose options:
  --filter <keyword>      Only download images containing the keyword

Other:
  --help                  Show this help
  --version               Show version
`

const downloadUsageText = `Usage: skopeo-cli download <image> [options]

Download a single Docker image to an oci-archive tarball.

Options:
  --save <path>           Save directory (default: ~/Downloads)
  --platform <os/arch>    Target platform (default: all)
  --registry <host>       Target registry (default: docker.senjone.com)
  --overwrite             Overwrite existing files
  --no-upload-script      Skip generating the upload script
`

const composeUsageText = `Usage: skopeo-cli compose <file> [options]

Batch download images referenced by a docker-compose file.

Options:
  --save <path>           Save directory (default: ~/Downloads)
  --platform <os/arch>    Target platform (default: all)
  --filter <keyword>      Only download images containing the keyword
  --registry <host>       Target registry (default: docker.senjone.com)
  --overwrite             Overwrite existing files
  --no-upload-script      Skip generating the upload script
`

func printUsage() {
	fmt.Print(usageText)
}

func printDownloadUsage() {
	fmt.Print(downloadUsageText)
}

func printComposeUsage() {
	fmt.Print(composeUsageText)
}

var errUsage = errors.New("usage error")

func downloadCommand(positional []string, opts options) error {
	if len(positional) == 0 {
		printDownloadUsage()
		return errUsage
	}
	image := positional[0]

	savePath := opts.savePath
	if savePath == "" {
		savePath = cli.DefaultDownloadPath()
	}
	outputDir := filepath.Join(savePath, "docker-image-will-upload")
	ensureDir(outputDir)

	if err := requireSkopeo(outputDir); err != nil {
		return err
	}

	result := skopeo.DownloadImage(image, skopeo.DownloadOptions{
		SavePath:  outputDir,
		Platform:  opts.platform,
		Overwrite: opts.overwrite,
		Registry:  opts.registry,
	})

	if !result.Success {
		return fmt.Errorf("download failed: %s", image)
	}
	console.ColorLog(fmt.Sprintf("Recorded: %s -> %s/%s", image, opts.registry, result.RepoPath), "green")
	if !opts.noUploadScript {
		upload.GenerateUploadScript([]skopeo.DownloadResult{result}, outputDir, opts.registry)
	}

	console.ColorLog(fmt.Sprintf("Output directory: %s", outputDir), "cyan")
	return nil
}

func composeCommand(positional []string, opts options) error {
	if len(positional) == 0 {
		printComposeUsage()
		return errUsage
	}
	file := positional[0]

	console.ColorLog(fmt.Sprintf("Parsing %s ...", file), "cyan")
	if opts.filter != "" {
		console.ColorLog(fmt.Sprintf("Applying filter: *%s*", opts.filter), "magenta")
	}

	images, err := compose.ParseComposeFile(file, opts.filter)
	if err != nil {
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

	if err := requireSkopeo(outputDir); err != nil {
		return err
	}

	console.ColorLog(fmt.Sprintf("Found %d matching images, starting download...", len(images)), "cyan")
	fmt.Println("==================================================")

	results := make([]skopeo.DownloadResult, 0, len(images))
	for i, img := range images {
		console.ColorLog(fmt.Sprintf("[%d/%d] Processing: %s", i+1, len(images), img), "cyan")
		result := skopeo.DownloadImage(img, skopeo.DownloadOptions{
			SavePath:  outputDir,
			Platform:  opts.platform,
			Overwrite: opts.overwrite,
			Registry:  opts.registry,
		})
		results = append(results, result)
		if !result.Success {
			console.ColorLogErr(fmt.Sprintf("Warning: %s download failed.", img), "yellow")
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
		upload.GenerateUploadScript(results, outputDir, opts.registry)
	}

	console.ColorLog(fmt.Sprintf("Output directory: %s", outputDir), "cyan")

	if failed > 0 {
		return fmt.Errorf("%d image(s) failed to download", failed)
	}
	return nil
}

func requireSkopeo(outputDir string) error {
	skopeoPath, err := skopeo.FindSkopeo()
	if err != nil {
		console.ColorLogErr("Error: skopeo not found. Please install skopeo first.", "red")
		console.ColorLogErr("  Windows: https://github.com/containers/skopeo/blob/main/install.md", "white")
		console.ColorLogErr("  Linux:   sudo apt install skopeo / sudo yum install skopeo", "white")
		return err
	}
	console.ColorLog(fmt.Sprintf("Found skopeo: %s", skopeoPath), "cyan")

	if v, verr := skopeo.VersionString(skopeoPath); verr == nil {
		if !skopeo.IsVersionSupported(v) {
			console.ColorLogErr(fmt.Sprintf("Error: skopeo version %s is too old; minimum supported is 1.6.0 (requires --all multi-architecture support).", v), "red")
			console.ColorLogErr("Versions below 1.6.0 may fail with:", "white")
			console.ColorLogErr("  unsupported MIME type for compression: application/vnd.in-toto+json", "white")
			return fmt.Errorf("skopeo version %s is below the minimum 1.6.0", v)
		}
		console.ColorLog(fmt.Sprintf("skopeo version: %s", v), "cyan")
	} else {
		console.ColorLog(fmt.Sprintf("Warning: could not check skopeo version: %v", verr), "yellow")
	}

	copied, err := skopeo.CopySkopeoToDir(outputDir)
	if err != nil {
		return err
	}
	if len(copied) == 0 {
		console.ColorLog(fmt.Sprintf("Info: no whitelisted skopeo binary (%s) found next to %s; upload scripts will use skopeo from PATH.", strings.Join(skopeo.SkopeoBinaryWhitelist, ", "), skopeoPath), "cyan")
	} else {
		console.ColorLog(fmt.Sprintf("Bundled skopeo binary: %s", strings.Join(copied, ", ")), "cyan")
	}
	return nil
}

func ensureDir(path string) {
	_ = os.MkdirAll(path, 0755)
}
