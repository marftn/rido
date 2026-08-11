package log

import (
	"fmt"
	"os"
)

func Errorf(format string, args ...any) {
	errMsg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[ERROR] %s", errMsg)
}

func Error(msg ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", fmt.Sprintln(msg...))
}

func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format, args...)
}

func Info(msg ...any) {
	fmt.Fprintln(os.Stdout, msg...)
}

func Debug(msg ...any) {
	fmt.Fprintf(os.Stderr, "[DEBUG] %s", fmt.Sprintln(msg...))
}

func Warnf(format string, args ...any) {
	warnMsg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[WARNING] %s", warnMsg)
}
