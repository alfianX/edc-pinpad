package logger

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

type SplitLevelHandler struct {
	infoHandler  slog.Handler
	errorHandler slog.Handler
}

func (h *SplitLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.infoHandler.Enabled(ctx, level) || h.errorHandler.Enabled(ctx, level)
}

func (h *SplitLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return h.errorHandler.Handle(ctx, r)
	}
	return h.infoHandler.Handle(ctx, r)
}

func (h *SplitLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SplitLevelHandler{
		infoHandler:  h.infoHandler.WithAttrs(attrs),
		errorHandler: h.errorHandler.WithAttrs(attrs),
	}
}

func (h *SplitLevelHandler) WithGroup(name string) slog.Handler {
	return &SplitLevelHandler{
		infoHandler:  h.infoHandler.WithGroup(name),
		errorHandler: h.errorHandler.WithGroup(name),
	}
}

// Tambahkan parameter "prefix" agar bisa dipakai untuk "gateway" dan "pinpad"
func SetupLogger(prefix string) *slog.Logger {
	// ==================================================================
	// BLOK PENAMBAHAN: Deteksi Absolute Path (Cross-Platform / Service)
	// ==================================================================
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Gagal mendeteksi executable path: %v", err)
	}

	baseDir := filepath.Dir(execPath)
	logDir := filepath.Join(baseDir, "logs")

	// Pastikan folder logs dibuat
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logDir = filepath.Join(os.TempDir(), "SwitchingGatewayLogs")
		_ = os.MkdirAll(logDir, 0755)
	}

	// Buat string path yang utuh (Contoh: C:\...\logs\gateway-info-%Y...log)
	infoLogPattern := filepath.Join(logDir, prefix+"-info-%Y-%m-%d-%H%M%S.log")
	infoLogLink := filepath.Join(logDir, prefix+"-info-current.log")

	errorLogPattern := filepath.Join(logDir, prefix+"-error-%Y-%m-%d-%H%M%S.log")
	errorLogLink := filepath.Join(logDir, prefix+"-error-current.log")
	// ==================================================================

	// ==================================================================
	// LOGIKA ASLI KAMU (Tidak ada yang diubah dari setting rotatelogs)
	// ==================================================================
	infoWriter, err := rotatelogs.New(
		infoLogPattern,
		rotatelogs.WithLinkName(infoLogLink),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(30*24*time.Hour),
		rotatelogs.WithRotationSize(100*1024*1024),
	)
	if err != nil {
		log.Fatalf("Gagal inisialisasi info rotatelogs: %v", err)
	}

	errorWriter, err := rotatelogs.New(
		errorLogPattern,
		rotatelogs.WithLinkName(errorLogLink),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(30*24*time.Hour),
		rotatelogs.WithRotationSize(100*1024*1024),
	)
	if err != nil {
		log.Fatalf("Gagal inisialisasi error rotatelogs: %v", err)
	}

	infoMultiWriter := io.MultiWriter(infoWriter, os.Stdout)
	errorMultiWriter := io.MultiWriter(errorWriter, os.Stderr)

	infoJsonHandler := slog.NewJSONHandler(infoMultiWriter, &slog.HandlerOptions{Level: slog.LevelInfo})
	errorJsonHandler := slog.NewJSONHandler(errorMultiWriter, &slog.HandlerOptions{Level: slog.LevelError})

	splitHandler := &SplitLevelHandler{
		infoHandler:  infoJsonHandler,
		errorHandler: errorJsonHandler,
	}

	return slog.New(splitHandler)
}
