package logging

import (
	"io"
	"log/slog"
	"os"
)

func Init(service string) {
	path := os.Getenv("LOG_FILE")
	if path == "" {
		path = "/home/martin/Documents/aws-v4/dev/logs/" + service + ".log"
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		slog.Error("cannot open log file", "err", err)
		return
	}
	w := io.MultiWriter(os.Stdout, f)
	logger := slog.New(slog.NewJSONHandler(w, nil)).With("service", service)
	slog.SetDefault(logger) // also redirects the std "log" package
}