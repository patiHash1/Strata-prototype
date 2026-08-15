// Package logger provides a small, opinionated wrapper around the standard
// library's structured logging package (log/slog). It exists to give the
// whole application one consistent logging entrypoint with JSON output,
// while still supporting the classic Printf/Println/Fatal helpers.
package logger

import (
	"log/slog"
	"os"
)

// Default returns a package-level JSON logger writing to stderr.
// Production deployments can override the global logger by calling
// slog.SetDefault with a custom handler before invoking any of the
// helper functions below.
func Default() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, nil))
}

var std = Default()

// SetDefault replaces the package-level logger used by the helpers below.
func SetDefault(l *slog.Logger) {
	if l == nil {
		return
	}
	std = l
	slog.SetDefault(l)
}

// Info logs at level Info.
func Info(msg string, args ...any) {
	std.Info(msg, args...)
}

// Warn logs at level Warn.
func Warn(msg string, args ...any) {
	std.Warn(msg, args...)
}

// Error logs at level Error.
func Error(msg string, args ...any) {
	std.Error(msg, args...)
}

// Debug logs at level Debug.
func Debug(msg string, args ...any) {
	std.Debug(msg, args...)
}
