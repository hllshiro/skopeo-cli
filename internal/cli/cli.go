package cli

import (
	"os"
	"runtime"
	"strings"
)

// ParsedArgs is the result of parsing command-line arguments.
type ParsedArgs struct {
	Command    string
	Positional []string
	Options    map[string]string
}

// ParseArgs parses command-line arguments in the shape:
//
//	<command> [positional...] [--key value | --flag | --key=value]...
//
// The command is the first non-flag token. Flags may appear anywhere after
// it. Boolean flags are stored with the value "true" unless an explicit
// value is provided.
func ParseArgs(args []string) ParsedArgs {
	positional := []string{}
	options := map[string]string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			body := arg[2:]
			if body == "" {
				continue
			}
			if eq := strings.IndexByte(body, '='); eq >= 0 {
				options[body[:eq]] = body[eq+1:]
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				options[body] = args[i+1]
				i++
			} else {
				options[body] = "true"
			}
		} else {
			positional = append(positional, arg)
		}
	}

	command := ""
	if len(positional) > 0 {
		command = positional[0]
		positional = positional[1:]
	}

	return ParsedArgs{Command: command, Positional: positional, Options: options}
}

// HasFlag reports whether a boolean flag is present and not explicitly false.
func HasFlag(options map[string]string, name string) bool {
	v, ok := options[name]
	if !ok {
		return false
	}
	switch strings.ToLower(v) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

// OptionString returns the value of a string option, or "" if absent.
func OptionString(options map[string]string, name string) string {
	return options[name]
}

// OptionStringDefault returns the value of a string option, or def if absent
// or empty.
func OptionStringDefault(options map[string]string, name, def string) string {
	if v, ok := options[name]; ok && v != "" && v != "true" {
		return v
	}
	return def
}

// DefaultDownloadPath returns the platform download directory.
func DefaultDownloadPath() string {
	return defaultDownloadPath(runtime.GOOS)
}

func defaultDownloadPath(goos string) string {
	if goos == "windows" {
		if p := os.Getenv("USERPROFILE"); p != "" {
			return p + `\Downloads`
		}
	}
	if h := os.Getenv("HOME"); h != "" {
		return h + "/Downloads"
	}
	cwd, _ := os.Getwd()
	return cwd + "/Downloads"
}
