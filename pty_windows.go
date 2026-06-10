//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/UserExistsError/conpty"
)

// winPty wraps a ConPTY handle + reference to track goroutines.
type winPty struct {
	c     *conpty.ConPty
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (p *winPty) Read(b []byte) (int, error) {
	if p.c != nil {
		return p.c.Read(b)
	}
	return 0, io.EOF
}

func (p *winPty) Write(b []byte) (int, error) {
	if p.c != nil {
		return p.c.Write(b)
	}
	if p.stdin != nil {
		return p.stdin.Write(b)
	}
	return 0, io.EOF
}

func (p *winPty) Close() error {
	var errs []error
	if p.c != nil {
		if e := p.c.Close(); e != nil {
			errs = append(errs, e)
		}
	}
	if p.stdin != nil {
		if e := p.stdin.Close(); e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (p *winPty) Resize(rows, cols uint16) error {
	if p.c != nil {
		return p.c.Resize(int(cols), int(rows))
	}
	return nil
}

func startPty(cmd *exec.Cmd, cols, rows uint16) (ptyIO, error) {
	// Try ConPTY first
	p, err := startConPty(cmd, cols, rows)
	if err == nil {
		return p, nil
	}

	fmt.Printf("[PTY] ConPTY 启动失败，降级到 pipe 模式: %v\n", err)

	// Fallback: pipe-based I/O
	return startPipe(cmd)
}

func startConPty(cmd *exec.Cmd, cols, rows uint16) (ptyIO, error) {
	if !conpty.IsConPtyAvailable() {
		return nil, fmt.Errorf("ConPTY 不可用")
	}

	cmdLine := buildCmdLine(cmd)
	fmt.Printf("[PTY] Windows ConPTY cmdLine: %s workDir: %s\n", cmdLine, cmd.Dir)

	var env []string
	if cmd.Env != nil {
		env = cmd.Env
	} else {
		env = os.Environ()
	}

	// Catch panics from the conpty library
	var c *conpty.ConPty
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("ConPTY panic: %v", r)
			}
		}()
		c, err = conpty.Start(cmdLine,
			conpty.ConPtyDimensions(int(cols), int(rows)),
			conpty.ConPtyWorkDir(cmd.Dir),
			conpty.ConPtyEnv(env),
		)
	}()

	if err != nil {
		return nil, err
	}

	fmt.Printf("[PTY] ConPTY 启动成功\n")
	return &winPty{c: c}, nil
}

// startPipe creates a pipe-based pseudo-terminal (no TTY, but works).
func startPipe(cmd *exec.Cmd) (ptyIO, error) {
	fmt.Printf("[PTY] 使用 pipe 模式: %s\n", cmd.Path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动进程: %w", err)
	}

	fmt.Printf("[PTY] Pipe 模式已启动 pid=%d\n", cmd.Process.Pid)

	pr, pw := io.Pipe()

	// Merge stdout + stderr concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(pw, stdout) }()
	go func() { defer wg.Done(); io.Copy(pw, stderr) }()
	go func() { wg.Wait(); pw.Close() }()

	return &pipePty{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    pr,
		closeOnce: sync.Once{},
	}, nil
}

// pipePty implements ptyIO using os/exec pipes.
type pipePty struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.Reader
	closeOnce sync.Once
}

func (p *pipePty) Read(b []byte) (int, error) {
	return p.stdout.Read(b)
}

func (p *pipePty) Write(b []byte) (int, error) {
	return p.stdin.Write(b)
}

func (p *pipePty) Close() error {
	var err error
	p.closeOnce.Do(func() {
		if p.stdin != nil {
			p.stdin.Close()
		}
		if p.cmd != nil && p.cmd.Process != nil {
			err = p.cmd.Process.Kill()
		}
	})
	return err
}

func (p *pipePty) Resize(rows, cols uint16) error {
	return nil
}

// buildCmdLine builds a Windows command line string from exec.Cmd.
func buildCmdLine(cmd *exec.Cmd) string {
	if len(cmd.Args) == 0 {
		return windowsCmdQuote(cmd.Path)
	}
	parts := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		parts[i] = windowsCmdQuote(arg)
	}
	return strings.Join(parts, " ")
}

// windowsCmdQuote quotes a string for the Windows command line if needed.
func windowsCmdQuote(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\n\v\"") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
