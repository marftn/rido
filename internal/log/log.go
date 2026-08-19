package log

import (
	"fmt"
	"os"
	"strings"

	"github.com/marftn/rido/internal/build"
)

func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s", formatLine(format, args...))
}

func Error(msg ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s", fmt.Sprintln(msg...))
}

func Infof(format string, args ...any) {
	fmt.Fprint(os.Stdout, formatLine(format, args...))
}

func Info(msg ...any) {
	fmt.Fprintln(os.Stdout, msg...)
}

func Debug(msg ...any) {
	if !build.IsDebug {
		return
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] %s", fmt.Sprintln(msg...))
}

func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARNING] %s", formatLine(format, args...))
}

// formatLine formats a message and adds a new line character if it was missing.
func formatLine(format string, args ...any) string {
	msg := fmt.Sprintf(format, args...)
	if strings.HasSuffix(msg, "\n") {
		return msg
	}

	return msg + "\n"
}
