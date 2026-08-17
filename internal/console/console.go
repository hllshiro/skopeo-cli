package console

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var colors = map[string]string{
	"red":     "\x1b[31m",
	"green":   "\x1b[32m",
	"yellow":  "\x1b[33m",
	"blue":    "\x1b[34m",
	"magenta": "\x1b[35m",
	"cyan":    "\x1b[36m",
	"white":   "\x1b[37m",
	"reset":   "\x1b[0m",
}

// ColorLog prints msg to stdout in the given color.
func ColorLog(msg, color string) {
	colorLogTo(os.Stdout, msg, color)
}

// ColorLogErr prints msg to stderr in the given color. Use it for error
// output so that normal output and errors can be separated in pipelines.
func ColorLogErr(msg, color string) {
	colorLogTo(os.Stderr, msg, color)
}

func colorLogTo(w io.Writer, msg, color string) {
	code := colors[strings.ToLower(color)]
	if code == "" {
		code = colors["white"]
	}
	fmt.Fprintf(w, "%s%s%s\n", code, msg, colors["reset"])
}
