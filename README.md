# SecToolBox

SecToolBox is a Wails-based desktop launcher for security tools. It helps you organize GUI tools, Java tools, Python scripts, and command-line executables in one local toolbox, with category management, environment configuration, and an integrated PTY terminal for interactive CLI workflows.

![Wails](https://img.shields.io/badge/Wails-v2.11.0-red)
![Vue](https://img.shields.io/badge/Vue-3.x-42b883)
![TypeScript](https://img.shields.io/badge/TypeScript-4.x-3178c6)
![Go](https://img.shields.io/badge/Go-1.23-00add8)

## Features

- Manage common security tools in a categorized toolbox.
- Add, edit, delete, filter, and launch tools from a desktop UI.
- Support multiple tool types:
  - Java GUI: launch with selected Java runtime.
  - Java CLI: open in system terminal or integrated terminal.
  - Python tools: detect virtual environments and activate them for PTY sessions.
  - EXE GUI: launch directly, with optional Windows elevation.
  - EXE CLI: open in system terminal or integrated terminal.
- Configure Java runtime versions and Python runtime path.
- Built-in xterm.js terminal backed by WebSocket + PTY:
  - ANSI color and cursor control support.
  - Interactive input and output.
  - Proper handling for progress bars and full-screen terminal behavior.
  - Terminal resize synchronization.
- Open CLI tools in either the system terminal or the integrated terminal.

## Screenshots

Main Window

![Main Window](docs/images/main_window.png)

Edit Env

![Edit Env](docs/images/edit_env.png)

Edit Tool Category

![Edit Tool Category](docs/images/edit_tool_category.png)

## Tech Stack

- Backend: Go, Wails v2
- Frontend: Vue 3, TypeScript, Vite, Naive UI
- Terminal: xterm.js, WebSocket, ConPTY on Windows, `creack/pty` on Unix-like systems
- Config persistence: YAML

## Requirements

- Go 1.23 or newer
- Node.js and npm
- Wails CLI

Install Wails if you do not already have it:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Check your local environment:

```bash
wails doctor
```

## Development

Clone the repository:

```bash
git clone https://github.com/Gokr-ble/SecToolBox.git
cd SecToolBox
```

Install frontend dependencies:

```bash
cd frontend
npm install
cd ..
```

Run in live development mode:

```bash
wails dev
```

This starts the Wails backend and Vite frontend with hot reload.

## Build

Create a production build:

```bash
wails build
```

The output binary is controlled by `wails.json`:

```json
{
  "name": "SecToolBox",
  "outputfilename": "SecToolBox"
}
```

## Usage

1. Start SecToolBox.
2. Add tool categories, such as `Web`, `Reverse`, `Exploit`, or `Forensics`.
3. Add tools with name, type, path, category, and optional description.
4. Configure Java runtime versions and Python path in the environment settings.
5. Launch tools:
   - Use `启动` to launch GUI tools or open CLI tools in the system terminal.
   - Use `终端` to start an integrated PTY terminal for CLI/Python tools.

## Tool Types

| Type | Meaning | Behavior |
| --- | --- | --- |
| `java-gui` | Java GUI tool | Runs `java -jar <tool>` directly |
| `java-cli` | Java CLI tool | Opens in terminal |
| `python` | Python tool/script/project | Opens in terminal, supports virtualenv detection |
| `exe-gui` | Windows GUI executable | Runs executable directly |
| `exe-cli` | Command-line executable | Opens in terminal |

## Python Virtual Environments

For Python tools, the integrated terminal can detect virtual environments near the tool path. It checks common directory names such as:

- `venv`
- `.venv`
- `env`
- `.env`
- `virtualenv`

When a virtual environment is selected, SecToolBox sets:

- `VIRTUAL_ENV`
- `PATH`/`Path` with the virtual environment `Scripts` or `bin` directory first
- Windows prompt prefix `(venv)`

You can verify activation in the integrated terminal:

```cmd
where python
python -c "import sys;print(sys.prefix)"
```

## Java Runtime Configuration

Java versions are configured in the environment settings. A Java entry can point to either a Java home directory or a `java.exe`/`java` binary. CLI and GUI Java tools can then select the expected runtime version.

## Config File

SecToolBox stores configuration in `config.yaml` in the working directory.

Example:

```yaml
version: "1.0"
categories:
  - Web
  - Reverse
tools:
  - id: "ffuf"
    name: "ffuf"
    type: "exe-cli"
    path: "D:\\PTE\\CLI\\ffuf.exe"
    description: "Fast web fuzzer"
    category: "Web"
  - id: "dirsearch"
    name: "dirsearch"
    type: "python"
    path: "D:\\PTE\\PTETools\\dirsearch\\dirsearch.py"
    description: "Web path scanner"
    category: "Web"
env:
  java:
    - "8": "C:\\Java\\jdk8\\bin\\java.exe"
    - "17": "C:\\Java\\jdk17\\bin\\java.exe"
  python: "C:\\Python312\\python.exe"
```

## Integrated Terminal Notes

The integrated terminal is designed for real CLI interaction. It uses xterm.js on the frontend and a local WebSocket bridge to a backend PTY session.

Platform behavior:

- Windows: uses ConPTY when available.
- Linux/macOS: uses `github.com/creack/pty`.

If a tool behaves differently from a normal terminal, verify that the selected working directory and environment variables are correct.

## Project Structure

```text
.
├── app.go                  # Wails-bound application methods
├── config_manager.go       # YAML config load/save
├── main.go                 # Wails entrypoint
├── process_runner.go       # System-terminal and GUI launch logic
├── pty_terminal.go         # PTY session and WebSocket server
├── pty_windows.go          # Windows ConPTY implementation
├── pty_unix.go             # Unix PTY implementation
├── frontend/
│   ├── src/
│   │   ├── App.vue
│   │   └── components/
│   │       ├── Main.vue
│   │       └── CliTerminal.vue
│   └── package.json
└── wails.json
```

## Security Notice

SecToolBox launches local tools and can pass environment variables to child processes. Only add tools from trusted locations, and review paths and runtime configuration before launching third-party binaries or scripts.

## Roadmap Ideas

- Import/export tool profiles.
- Per-tool custom working directory and environment variables.
- Release packaging with signed binaries.
- Optional screenshots and demo GIFs for GitHub Releases.

## License

No license file is included yet. Add a `LICENSE` file before publishing if you want to define reuse terms.
