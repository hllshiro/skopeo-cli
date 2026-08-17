package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFileNames lists the standard compose file names in precedence order,
// used when the argument is a directory or an exact file is not found.
var ComposeFileNames = []string{
	"compose.yaml",
	"compose.yml",
	"docker-compose.yaml",
	"docker-compose.yml",
}

// ResolveComposeFile returns the path of the compose file to parse. If file
// exists and is not a directory it is returned as-is; otherwise file is
// treated as a directory and the standard compose file names are tried inside
// it. A non-existent file that is not a directory yields an error.
func ResolveComposeFile(file string) (string, error) {
	if fi, err := os.Stat(file); err == nil && !fi.IsDir() {
		return file, nil
	}
	for _, name := range ComposeFileNames {
		cand := filepath.Join(file, name)
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("compose file not found: %s", file)
}

// ParseComposeFile extracts unique image references from a compose file. If
// filter is non-empty, only images containing the filter substring are
// returned (case-sensitive). Image values are interpolated with environment
// variables and a .env file next to the compose file, supporting ${VAR},
// ${VAR:-default} and ${VAR-default} forms.
func ParseComposeFile(filePath, filter string) ([]string, error) {
	path, err := ResolveComposeFile(filePath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read compose file %s: %w", path, err)
	}

	images, err := parseComposeImages(data, envLookup(filepath.Dir(path)))
	if err != nil {
		return nil, err
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

// parseComposeImages walks the YAML tree and collects services.*.image values
// in document order, deduplicating. Each value is interpolated via lookup.
func parseComposeImages(data []byte, lookup func(string) (string, bool)) ([]string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid compose file: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, nil
	}
	services := mappingValue(root, "services")
	if services == nil || services.Kind != yaml.MappingNode {
		return nil, nil
	}

	images := []string{}
	seen := make(map[string]bool)
	for i := 0; i+1 < len(services.Content); i += 2 {
		val := services.Content[i+1]
		if val.Kind != yaml.MappingNode {
			continue
		}
		imgNode := mappingValue(val, "image")
		if imgNode == nil || imgNode.Kind != yaml.ScalarNode || imgNode.Tag != "!!str" {
			continue
		}
		img := strings.TrimSpace(interpolate(imgNode.Value, lookup))
		if img != "" && !seen[img] {
			seen[img] = true
			images = append(images, img)
		}
	}
	return images, nil
}

// mappingValue returns the value node for key in a mapping node, or nil.
// YAML merge keys (<<) are resolved first: merged sources are applied before
// explicit keys, so explicit keys win, matching YAML merge semantics.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	merged := map[string]*yaml.Node{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if k.Value == "<<" {
			for _, src := range mergeSources(v) {
				for j := 0; j+1 < len(src.Content); j += 2 {
					merged[src.Content[j].Value] = src.Content[j+1]
				}
			}
		}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Value != "<<" {
			merged[k.Value] = node.Content[i+1]
		}
	}
	return merged[key]
}

// mergeSources expands the value of a << key into source mapping nodes,
// following aliases and sequences of mappings.
func mergeSources(v *yaml.Node) []*yaml.Node {
	if v == nil {
		return nil
	}
	if v.Kind == yaml.AliasNode {
		return mergeSources(v.Alias)
	}
	if v.Kind == yaml.MappingNode {
		return []*yaml.Node{v}
	}
	if v.Kind == yaml.SequenceNode {
		var out []*yaml.Node
		for _, item := range v.Content {
			out = append(out, mergeSources(item)...)
		}
		return out
	}
	return nil
}

// composeVarRe matches $$ escapes, ${...} expressions and bare $VAR forms.
var composeVarRe = regexp.MustCompile(`\$\$|\$\{[^{}]*\}|\$[A-Za-z_][A-Za-z0-9_]*`)

// interpolate replaces ${VAR}, ${VAR:-default}, ${VAR-default} and $VAR in s
// using lookup. $$ is unescaped to a literal $.
func interpolate(s string, lookup func(string) (string, bool)) string {
	return composeVarRe.ReplaceAllStringFunc(s, func(m string) string {
		if m == "$$" {
			return "$"
		}
		name, def := m, ""
		hasDef, emptyIsUnset := false, false
		if strings.HasPrefix(m, "${") {
			name, def, hasDef, emptyIsUnset = splitVar(m[2 : len(m)-1])
		} else {
			name = m[1:]
		}
		if v, ok := lookup(name); ok && !(emptyIsUnset && v == "") {
			return v
		}
		if hasDef {
			return def
		}
		return ""
	})
}

// splitVar parses the inner text of ${...} into a variable name and optional
// default. Supported modifiers: ${VAR:-default} (default when unset or empty)
// and ${VAR-default} (default only when unset). Unsupported modifiers such as
// ${VAR:?msg} are treated as name-only references.
func splitVar(inner string) (name, def string, hasDef, emptyIsUnset bool) {
	i := 0
	for i < len(inner) && isVarNameByte(inner[i]) {
		i++
	}
	name = inner[:i]
	rest := inner[i:]
	switch {
	case strings.HasPrefix(rest, ":-"):
		return name, rest[2:], true, true
	case strings.HasPrefix(rest, "-"):
		return name, rest[1:], true, false
	default:
		return name, "", false, false
	}
}

func isVarNameByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// envLookup builds a lookup function that resolves variables from the process
// environment first, then from a .env file in dir (KEY=value lines, # comments,
// surrounding quotes stripped).
func envLookup(dir string) func(string) (string, bool) {
	vals := map[string]string{}
	if data, err := os.ReadFile(filepath.Join(dir, ".env")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				continue
			}
			key := strings.TrimSpace(line[:eq])
			val := strings.TrimSpace(line[eq+1:])
			if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
				val = val[1 : len(val)-1]
			}
			vals[key] = val
		}
	}
	return func(key string) (string, bool) {
		if v, ok := os.LookupEnv(key); ok {
			return v, true
		}
		v, ok := vals[key]
		return v, ok
	}
}
