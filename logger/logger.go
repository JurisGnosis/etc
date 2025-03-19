package logger

import (
	"log/slog"
	"runtime"
)

// ErrorWithLine 记录错误信息并包含文件名和行号
func ErrorWithLine(msg string, args ...any) {
	_, file, line, _ := runtime.Caller(1)
	newArgs := append([]any{"file", file, "line", line}, args...)
	slog.Error(msg, newArgs...)
}

// InfoWithLine 记录信息并包含文件名和行号
func InfoWithLine(msg string, args ...any) {
	_, file, line, _ := runtime.Caller(1)
	newArgs := append([]any{"file", file, "line", line}, args...)
	slog.Info(msg, newArgs...)
}

// WarnWithLine 记录警告信息并包含文件名和行号
func WarnWithLine(msg string, args ...any) {
	_, file, line, _ := runtime.Caller(1)
	newArgs := append([]any{"file", file, "line", line}, args...)
	slog.Warn(msg, newArgs...)
}
