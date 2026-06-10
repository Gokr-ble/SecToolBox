package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

type ProcessRunner struct {
	processes map[string]*exec.Cmd
	mutex     sync.Mutex
	config    *ConfigManager
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{
		processes: make(map[string]*exec.Cmd),
		config:    NewConfigManager(),
	}
}

func (pr *ProcessRunner) StartTool(tool ToolConfig, args string, priviledged bool) (string, error) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	var cmd *exec.Cmd
	ctx := context.Background()

	switch tool.Type {
	case "java-gui":
		javaHome := pr.getJavaHome(tool.JavaVersion)
		var javaExec string
		if runtime.GOOS == "windows" {
			javaExec = filepath.Join(javaHome, "bin", "java.exe")
		} else {
			javaExec = filepath.Join(javaHome, "bin", "java")
		}
		fullArgs := []string{"-jar", tool.Path}
		if args != "" {
			fullArgs = append(fullArgs, strings.Fields(args)...)
		}

		cmd = exec.CommandContext(ctx, javaExec, fullArgs...)
		cmd.Dir = filepath.Dir(tool.Path)

		fmt.Println("Command:", cmd.String())

	case "java-cli":
		return pr.openTerminal(tool)

	case "python":
		return pr.openTerminal(tool)

	case "exe-gui":
		cmd = exec.CommandContext(ctx, tool.Path)
		if args != "" {
			cmd.Args = append(cmd.Args, strings.Fields(args)...)
		}
		cmd.Dir = filepath.Dir(tool.Path)

		if (priviledged) && runtime.GOOS == "windows" && !isAdmin() {
			fmt.Println("尝试管理员权限运行")
			result, err := runAsAdmin(tool.Path, cmd.Dir, args)
			return result, err
		}

	case "exe-cli":
		return pr.openTerminal(tool)
	}

	// 设置工作目录和环境变量
	// if tool.WorkingDir != "" {
	//     cmd.Dir = tool.WorkingDir
	// }

	// for _, envVar := range tool.EnvVars {
	//     parts := strings.SplitN(envVar, "=", 2)
	//     if len(parts) == 2 {
	//         cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", parts[0], parts[1]))
	//     }
	// }

	// 对于GUI工具，直接启动
	if strings.HasSuffix(tool.Type, "-gui") {
		err := cmd.Start()
		if err != nil {
			fmt.Printf("启动失败: %v\n", err)
			return "", err
		}
		pr.processes[tool.ID] = cmd
		return fmt.Sprintf("启动成功 (PID: %d)", cmd.Process.Pid), nil
	}

	// 对于CLI工具，捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("启动失败: %v\n", err)
		return string(output), err
	}

	return string(output), nil
}

func (pr *ProcessRunner) openTerminal(tool ToolConfig) (string, error) {
	workDir := tool.Path
	if !isDir(workDir) {
		workDir = filepath.Dir(workDir)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		userProfile := os.Getenv("USERPROFILE")
		wtPath := filepath.Join(userProfile, "AppData", "Local", "Microsoft", "WindowsApps", "wt.exe")
		if _, err := os.Stat(wtPath); err == nil {
			cmd = exec.Command(wtPath, "nt", "-d", workDir)
			fmt.Println("Command:", cmd.String())
		} else {
			cmd = exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/K", "cd /d "+workDir)
			fmt.Println("Command:", cmd.String())
		}
	case "darwin":
		cmd = exec.Command("open", "-a", "Terminal", workDir)
		fmt.Println("Command:", cmd.String())
	case "linux":
		cmd = exec.Command("x-terminal-emulator", "--working-directory", workDir)
		fmt.Println("Command:", cmd.String())
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf("终端启动失败: %v\n", err)
		return "", err
	}

	return "终端已打开", nil
}

func (pr *ProcessRunner) getJavaHome(version string) string {
	// 从配置文件中读取对应版本的Java路径
	env, err := pr.config.GetEnvConfig()
	if err != nil {
		// 如果读取配置失败，回退到环境变量方式
		return os.Getenv(fmt.Sprintf("JAVA%s_HOME", version))
	}

	// 查找指定版本的Java路径
	for _, javaMap := range env.Java {
		if javaMap[version] != "" {
			// 返回Java可执行文件的目录路径（去掉bin/java.exe部分）
			javaPath := javaMap[version]
			if runtime.GOOS == "windows" {
				if strings.HasSuffix(javaPath, "java.exe") {
					return filepath.Dir(filepath.Dir(javaPath))
				}
			} else {
				if strings.HasSuffix(javaPath, "java") {
					return filepath.Dir(filepath.Dir(javaPath))
				}
			}
			return javaPath
		}
	}

	// 如果没有找到指定版本，回退到环境变量方式
	return os.Getenv(fmt.Sprintf("JAVA%s_HOME", version))
}

func (pr *ProcessRunner) StopTool(toolID string) error {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	if cmd, exists := pr.processes[toolID]; exists {
		err := cmd.Process.Kill()
		delete(pr.processes, toolID)
		return err
	}

	return fmt.Errorf("未找到该工具的运行实例")
}

func runAsAdmin(exePath, cwd, args string) (string, error) {
	if !isAdmin() {
		verb := "runas"
		// exePath, _ := os.Executable()
		// cwd, _ := os.Getwd()
		// args := strings.Join(os.Args[1:], " ")

		fmt.Println("[*] exePath", exePath)
		fmt.Println("[*] cwd", cwd)
		fmt.Println("[*] args", args)

		verbPtr, _ := syscall.UTF16PtrFromString(verb)
		exePtr, _ := syscall.UTF16PtrFromString(exePath)
		cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
		argsPtr, _ := syscall.UTF16PtrFromString(args)

		var showCmd int32 = 1 // SW_NORMAL

		err := windows.ShellExecute(0, verbPtr, exePtr, argsPtr, cwdPtr, showCmd)
		if err != nil && err.Error() != "The operation completed successfully." {
			fmt.Println("提升权限失败:", err)
			return "提升权限失败", err
		}

	}
	return "", nil
}

func isAdmin() bool {
	// if runtime.GOOS == "windows" {
	// 	// Windows下检查管理员权限的简单方法
	// 	cmd := exec.Command("net", "session")
	// 	err := cmd.Run()
	// 	return err == nil
	// }
	return windows.GetCurrentProcessToken().IsElevated()
}
