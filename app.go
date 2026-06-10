package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	config       *ConfigManager
	runner       *ProcessRunner
	ptyTerminal  *PtyTerminalManager
}

func NewApp() *App {
	mgr := NewPtyTerminalManager(NewConfigManager())
	if err := mgr.StartServer(0); err != nil {
		log.Fatal("启动 WebSocket 服务器失败:", err)
	}

	app := &App{
		config:      NewConfigManager(),
		runner:      NewProcessRunner(),
		ptyTerminal: mgr,
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.ptyTerminal.Shutdown()
}

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetTools 获取所有工具配置
func (a *App) GetTools() ([]ToolConfig, error) {
	return a.config.GetTools()
}

// SaveTools 保存工具配置
func (a *App) SaveTools(tools []ToolConfig) error {
	return a.config.SaveTools(tools)
}

// StartTool 启动工具
func (a *App) StartTool(toolID string, priviledged bool) (string, error) {
	tools, err := a.config.GetTools()
	if err != nil {
		return "", err
	}

	for _, tool := range tools {
		if tool.ID == toolID {
			return a.runner.StartTool(tool, "", priviledged)
		}
	}

	return "", fmt.Errorf("未找到ID为 %s 的工具", toolID)
}

func (a *App) GetEnvConfig() (*EnvConfig, error) {
	return a.config.GetEnvConfig()
}

func (a *App) SaveEnvConfig(env *EnvConfig) error {
	return a.config.SaveEnvConfig(env)
}

func (a *App) GetCategories() ([]string, error) {
	return a.config.GetCategories()
}

func (a *App) SaveCategories(categories []string) error {
	return a.config.SaveCategories(categories)
}

// OpenFileDialog 打开文件选择对话框
func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工具可执行文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "可执行文件",
				Pattern:     "*.exe;*.jar;*.py",
			},
		},
	})
}

// OpenDirectoryDialog 打开目录选择对话框
func (a *App) OpenDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择工作目录",
	})
}

// RunPtyTool 启动 PTY 终端会话，返回 sessionID 和 WebSocket 端口
func (a *App) RunPtyTool(toolID string, javaVersion string, venvPath string) (map[string]interface{}, error) {
	tools, err := a.config.GetTools()
	if err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if tool.ID == toolID {
			sid, err := a.ptyTerminal.StartPtySession(tool, javaVersion, venvPath)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"sessionID": sid,
				"wsPort":    a.ptyTerminal.Port(),
			}, nil
		}
	}

	return nil, fmt.Errorf("未找到ID为 %s 的工具", toolID)
}

// StopPtySession 停止 PTY 终端会话
func (a *App) StopPtySession(sessionID string) error {
	return a.ptyTerminal.StopPtySession(sessionID)
}

// DetectVenvs 检测目录下的Python虚拟环境
func (a *App) DetectVenvs(dirPath string) []string {
	return a.ptyTerminal.DetectVenvs(dirPath)
}
