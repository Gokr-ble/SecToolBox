//go:build !windows

package main

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

type unixPty struct {
	f *os.File
}

func (p *unixPty) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPty) Write(b []byte) (int, error) { return p.f.Write(b) }
func (p *unixPty) Close() error                { return p.f.Close() }
func (p *unixPty) Resize(rows, cols uint16) error {
	return pty.Setsize(p.f, &pty.Winsize{Rows: rows, Cols: cols})
}

func startPty(cmd *exec.Cmd, cols, rows uint16) (ptyIO, error) {
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return nil, err
	}
	return &unixPty{f: f}, nil
}
