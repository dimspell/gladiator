package redirect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// LineReader reads lines from stdin and writes to an io.Writer.
type LineReader struct {
	logger *slog.Logger
}

// NewLineReader creates a new LineReader instance.
func NewLineReader(_ Mode, _ *Addressing) (Redirect, error) {
	logger := slog.With(slog.String("component", "line-reader"))
	return &LineReader{logger: logger}, nil
}

// Run reads from stdin and writes to the provided io.Writer.
func (p *LineReader) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	scanner := bufio.NewScanner(os.Stdin)

	p.logger.Info("LineReader started, waiting for input...")

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			p.logger.Info("LineReader shutting down due to context cancellation")
			return ctx.Err()
		default:
			line := scanner.Text()
			if err := onReceive([]byte(line + "\n")); err != nil {
				p.logger.Error("Failed to write line", "error", err)
				return fmt.Errorf("line-reader: failed to write output: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		p.logger.Error("Error reading input", "error", err)
		return fmt.Errorf("line-reader: error reading input: %w", err)
	}

	p.logger.Info("LineReader finished reading input")
	return nil
}

// Write outputs a message to stdout.
func (p *LineReader) Write(msg []byte) (int, error) {
	n, err := fmt.Fprintf(os.Stdout, "%s\n", msg)
	if err != nil {
		p.logger.Error("Failed to write to stdout", "error", err)
		return n, fmt.Errorf("line-reader: failed to write to stdout: %w", err)
	}
	return n, nil
}

// Close is a no-op for LineReader.
func (p *LineReader) Close() error {
	p.logger.Info("LineReader closed")
	return nil
}
