package redirect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
)

type LineReader struct{}

func NewLineReader(_ Mode, _ *Addressing) (Redirect, error) {
	return &LineReader{}, nil
}

func (p *LineReader) Run(_ context.Context, rw io.Writer) error {
	rd := bufio.NewReader(os.Stdin)
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return err
		}
		if _, err := rw.Write(line); err != nil {
			return err
		}
	}
}

func (p *LineReader) Write(msg []byte) (int, error) {
	return fmt.Fprintf(os.Stdout, "%s\n", msg)
}

func (p *LineReader) Close() error {
	return nil
}
