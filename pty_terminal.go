package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ptyIO abstracts platform-specific PTY I/O.
type ptyIO interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	Resize(rows, cols uint16) error
}

type PtySession struct {
	ID   string
	Cmd  *exec.Cmd
	Pty  ptyIO
	Conn *websocket.Conn
	done chan struct{}
	close sync.Once
}

type PtyTerminalManager struct {
	sessions map[string]*PtySession
	mu       sync.Mutex
	config   *ConfigManager
	wsPort   int
	server   *http.Server
}

func NewPtyTerminalManager(config *ConfigManager) *PtyTerminalManager {
	return &PtyTerminalManager{
		sessions: make(map[string]*PtySession),
		config:   config,
	}
}

func (m *PtyTerminalManager) StartServer(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", m.handleWebSocket)

	m.server = &http.Server{Handler: mux}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}

	m.wsPort = ln.Addr().(*net.TCPAddr).Port
	fmt.Printf("[PTY] WebSocket 服务器启动 127.0.0.1:%d\n", m.wsPort)

	go m.server.Serve(ln)
	return nil
}

func (m *PtyTerminalManager) Port() int {
	return m.wsPort
}

func (m *PtyTerminalManager) Shutdown() {
	if m.server != nil {
		m.server.Close()
	}
	m.mu.Lock()
	for _, s := range m.sessions {
		s.Close()
	}
	m.mu.Unlock()
}

func (m *PtyTerminalManager) StartPtySession(tool ToolConfig, javaVersion string, venvPath string) (string, error) {
	sessionID := fmt.Sprintf("pty-%d", nextSessionID.Add(1))
	fmt.Printf("[PTY] StartPtySession id=%s tool=%s type=%s path=%s venv=%s java=%s\n",
		sessionID, tool.Name, tool.Type, tool.Path, venvPath, javaVersion)

	workDir := toolWorkDir(tool)

	shell, shellArgs := getShell()
	cmd := exec.Command(shell, shellArgs...)
	cmd.Dir = workDir

	// Set environment: venv activation for Python tools
	if tool.Type == "python" && venvPath != "" {
		cmd.Env = buildPythonVenvEnv(venvPath)
	}

	fmt.Printf("[PTY] Shell: %s %v workDir=%s venv=%s\n", shell, shellArgs, workDir, venvPath)

	p, err := startPty(cmd, 120, 35)
	if err != nil {
		fmt.Printf("[PTY] startPty 失败: %v\n", err)
		return "", fmt.Errorf("启动 PTY 失败: %w", err)
	}
	fmt.Printf("[PTY] PTY 已启动 session=%s\n", sessionID)

	m.mu.Lock()
	m.sessions[sessionID] = &PtySession{
		ID:   sessionID,
		Cmd:  cmd,
		Pty:  p,
		done: make(chan struct{}),
	}
	m.mu.Unlock()

	return sessionID, nil
}

// getShell returns the default interactive shell for the platform.
func getShell() (string, []string) {
	if goruntime.GOOS == "windows" {
		return "cmd.exe", nil
	}
	return "bash", []string{"--login"}
}

// toolWorkDir returns the working directory for a tool.
func toolWorkDir(tool ToolConfig) string {
	if isDir(tool.Path) {
		return tool.Path
	}
	return filepath.Dir(tool.Path)
}

func buildPythonVenvEnv(venvPath string) []string {
	env := os.Environ()
	env = setEnvValue(env, "VIRTUAL_ENV", venvPath)
	env = unsetEnvValue(env, "PYTHONHOME")

	if goruntime.GOOS == "windows" {
		env = prependEnvPath(env, filepath.Join(venvPath, "Scripts"), ";")
		env = setEnvValue(env, "PROMPT", "(venv) $P$G")
	} else {
		env = prependEnvPath(env, filepath.Join(venvPath, "bin"), ":")
	}

	return env
}

func prependEnvPath(env []string, dir string, sep string) []string {
	for i, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(key, "PATH") {
			env[i] = key + "=" + dir + sep + value
			return removeDuplicateEnvKeys(env, key)
		}
	}
	return append(env, "PATH="+dir)
}

func setEnvValue(env []string, name string, value string) []string {
	for i, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(key, name) {
			env[i] = key + "=" + value
			return removeDuplicateEnvKeys(env, key)
		}
	}
	return append(env, name+"="+value)
}

func unsetEnvValue(env []string, name string) []string {
	filtered := env[:0]
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func removeDuplicateEnvKeys(env []string, canonicalKey string) []string {
	seenCanonical := false
	filtered := env[:0]
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, canonicalKey) {
			if seenCanonical {
				continue
			}
			seenCanonical = true
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

var nextSessionID atomic.Int64

func (m *PtyTerminalManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("sid")
	fmt.Printf("[PTY] WS 连接请求 sid=%s remote=%s\n", sid, r.RemoteAddr)

	m.mu.Lock()
	session, exists := m.sessions[sid]
	m.mu.Unlock()

	if !exists {
		fmt.Printf("[PTY] WS 会话不存在: %s\n", sid)
		http.Error(w, "session not found", 404)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("[PTY] WS upgrade 失败: %v\n", err)
		return
	}
	fmt.Printf("[PTY] WS 已升级 sid=%s\n", sid)

	session.Conn = conn

	// PTY → WebSocket
	go func() {
		defer fmt.Printf("[PTY] PTY→WS goroutine 退出 sid=%s\n", sid)

		buf := make([]byte, 4096)
		totalBytes := 0
		chunkCount := 0
		for {
			n, err := session.Pty.Read(buf)
			if err != nil {
				fmt.Printf("[PTY] PTY 读取结束 sid=%s err=%v totalBytes=%d chunks=%d\n",
					sid, err, totalBytes, chunkCount)
				break
			}
			totalBytes += n
			chunkCount++
			if chunkCount <= 3 {
				fmt.Printf("[PTY] PTY→WS #%d sid=%s n=%d: %q\n", chunkCount, sid, n, string(buf[:min(n, 80)]))
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				fmt.Printf("[PTY] WS 写入失败 sid=%s: %v\n", sid, err)
				break
			}
		}
		// Process exited
		exitMsg := []byte(`{"type":"exit","code":0}`)
		conn.WriteMessage(websocket.TextMessage, exitMsg)
		conn.Close()
		m.mu.Lock()
		delete(m.sessions, sid)
		m.mu.Unlock()
		fmt.Printf("[PTY] 会话已清理 sid=%s\n", sid)
	}()

	// WebSocket → PTY
	go func() {
		defer fmt.Printf("[PTY] WS→PTY goroutine 退出 sid=%s\n", sid)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("[PTY] WS 读取结束 sid=%s: %v\n", sid, err)
				break
			}

			if len(msg) > 0 && msg[0] == '{' {
				var ctrl struct {
					Type string `json:"type"`
					Cols uint16 `json:"cols"`
					Rows uint16 `json:"rows"`
				}
				if json.Unmarshal(msg, &ctrl) == nil && ctrl.Type == "resize" {
					fmt.Printf("[PTY] resize sid=%s cols=%d rows=%d\n", sid, ctrl.Cols, ctrl.Rows)
					session.Pty.Resize(ctrl.Rows, ctrl.Cols)
					continue
				}
			}

			session.Pty.Write(msg)
		}
	}()
}

func (m *PtyTerminalManager) StopPtySession(sessionID string) error {
	fmt.Printf("[PTY] StopPtySession sid=%s\n", sessionID)

	m.mu.Lock()
	session, exists := m.sessions[sessionID]
	m.mu.Unlock()

	if !exists {
		fmt.Printf("[PTY] StopPtySession 会话不存在: %s\n", sessionID)
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	session.Close()
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	fmt.Printf("[PTY] StopPtySession 完成 sid=%s\n", sessionID)
	return nil
}

func (s *PtySession) Close() {
	s.close.Do(func() {
		fmt.Printf("[PTY] Close session=%s\n", s.ID)
		if s.Conn != nil {
			fmt.Printf("[PTY] 关闭 WS 连接\n")
			s.Conn.Close()
		}
		if s.Pty != nil {
			fmt.Printf("[PTY] 关闭 PTY\n")
			s.Pty.Close()
		}
		fmt.Printf("[PTY] Close 完成 session=%s\n", s.ID)
	})
}

func (m *PtyTerminalManager) DetectVenvs(dirPath string) []string {
	var venvs []string
	venvNames := []string{"venv", ".venv", "env", ".env", "virtualenv"}

	var dirsToCheck []string
	current := dirPath
	dirsToCheck = append(dirsToCheck, current)
	for i := 0; i < 3; i++ {
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		dirsToCheck = append(dirsToCheck, parent)
		current = parent
	}

	seen := make(map[string]bool)
	for _, dir := range dirsToCheck {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			isVenv := false
			for _, vn := range venvNames {
				if name == vn {
					isVenv = true
					break
				}
			}
			if !isVenv {
				cfgPath := filepath.Join(dir, entry.Name(), "pyvenv.cfg")
				if _, err := os.Stat(cfgPath); err == nil {
					isVenv = true
				}
			}
			if isVenv {
				fullPath := filepath.Join(dir, entry.Name())
				if !seen[fullPath] {
					venvs = append(venvs, fullPath)
					seen[fullPath] = true
				}
			}
		}
	}

	return venvs
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
