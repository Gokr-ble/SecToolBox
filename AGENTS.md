# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## Build & Development

- **Live development**: `wails dev` — starts Vite dev server with hot reload and Go backend
- **Production build**: `wails build` — compiles Go binary with embedded frontend assets
- **Frontend only**: `cd frontend && npm run dev` (Vite), `npm run build` (type-check + bundle)
- **Project config**: `wails.json` — controls app name, output filename, frontend commands

## Architecture

A **Wails v2** desktop app (Go backend + Vue 3/TypeScript/Vite frontend) for managing and launching security tools. The Go runtime binds methods to the frontend via the Wails IPC bridge — the frontend calls Go functions as if they were local.

### Backend (root `*.go` files)

Four files in a flat `package main`:

- **`main.go`** — Entry point. Embeds `frontend/dist` via `//go:embed`, creates the Wails app (1200x800, title "SecToolBox"), binds `App` struct to frontend.
- **`app.go`** — `App` struct holding `ConfigManager` and `ProcessRunner`. All methods bound to Wails (`GetTools`, `SaveTools`, `StartTool`, `GetEnvConfig`, `SaveEnvConfig`, `OpenFileDialog`, `OpenDirectoryDialog`). `startup()` stores the Wails context for dialog APIs.
- **`config_manager.go`** — YAML config persistence. `AppConfig` holds `version`, `[]ToolConfig`, and `EnvConfig`. Reads/writes `config.yaml` in the working directory. Each tool has ID, name, type, path, category, and optional javaVersion/description.
- **`process_runner.go`** — Executes tools based on type:
  - `*-gui` types: launches directly (Java via `java -jar`, EXE directly). Supports Windows admin elevation via `ShellExecute` with `runas` verb.
  - `*-cli` types / `python`: opens a terminal window (Windows Terminal if available, falls back to cmd.exe).
  - Tracks running processes by tool ID for later termination.

### Frontend (`frontend/src/`)

- Vue 3 with `<script setup>` and TypeScript
- **Naive UI** component library for UI (NCard, NTable, NModal, NForm, NTabs, NDynamicInput, etc.)
- **`App.vue`** — root wrapper with NMessageProvider and NDialogProvider
- **`Main.vue`** — the entire application UI: tool list with category tabs, name filter, add/edit/delete modals, and environment config modal (Java versions + Python path). Uses auto-generated Wails bindings at `wailsjs/go/main/App` to call Go methods.
- **`style.css`** — global styles

### Tool types

| Type | Display | Launch behavior |
|------|---------|----------------|
| `java-gui` | Java GUI | `java -jar <path>` |
| `java-cli` | Java CLI | Terminal window |
| `python` | Python | Terminal window |
| `exe-gui` | EXE GUI | Direct launch (supports admin elevation) |
| `exe-cli` | EXE CLI | Terminal window |

### Config file (`config.yaml`)

```yaml
version: "1.0"
tools:
  - id: "..."
    name: "Tool Name"
    type: "exe-gui"
    path: "C:\\path\\to\\tool.exe"
    description: "..."
    category: "Web"
env:
  java:
    - "8": "C:\\path\\to\\java8\\bin\\java.exe"
    - "11": "C:\\path\\to\\java11\\bin\\java.exe"
  python: "C:\\path\\to\\python.exe"
```

## Key Dependencies

- **Go**: Wails v2.11.0, `golang.org/x/sys` (Windows API), `gopkg.in/yaml.v3`
- **Frontend**: Vue 3.2, Naive UI 2.42, Vite 3, TypeScript 4.6
