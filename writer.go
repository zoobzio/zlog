package zlog

import (
	"io"
	"os"
)

// Writer represents the output destination for the logger
type Writer interface {
	io.Writer
}

// writerConfig holds the current writer configuration
type writerConfig struct {
	writer Writer
}

var config = &writerConfig{
	writer: os.Stdout,
}

// SetWriter sets the output writer for the logger
func SetWriter(w Writer) {
	config.writer = w
}

// getWriter returns the current writer
func getWriter() Writer {
	return config.writer
}