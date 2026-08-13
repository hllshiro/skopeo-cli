package compose

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var imageLineRe = regexp.MustCompile(`^\s*image:\s*(\S+)`)

// ParseComposeFile extracts unique image references from a compose file. If
// filter is non-empty, only images containing the filter substring are
// returned (case-sensitive).
func ParseComposeFile(filePath, filter string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("File not found: %s", filePath)
	}

	var images []string
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		m := imageLineRe.FindStringSubmatch(line)
		if len(m) > 1 {
			img := strings.ReplaceAll(strings.ReplaceAll(m[1], `"`, ""), `'`, "")
			if !seen[img] {
				seen[img] = true
				images = append(images, img)
			}
		}
	}

	if filter != "" {
		filtered := []string{}
		for _, img := range images {
			if strings.Contains(img, filter) {
				filtered = append(filtered, img)
			}
		}
		return filtered, nil
	}

	return images, nil
}
