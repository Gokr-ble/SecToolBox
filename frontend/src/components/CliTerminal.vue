<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    :title="`终端 - ${tool.Name}`"
    style="width: 92vw; max-width: 1600px;"
    size="huge"
    :mask-closable="false"
    @after-leave="handleAfterLeave"
  >
    <template #header-extra>
      <n-space align="center">
        <n-select
          v-if="tool.Type === 'python'"
          v-model:value="selectedVenv"
          :options="venvOptions"
          placeholder="选择虚拟环境（可选）"
          clearable
          style="width: 200px; flex-shrink: 0;"
        />
        <n-select
          v-if="tool.Type === 'java-cli'"
          v-model:value="selectedJavaVersion"
          :options="javaVersionSelectOptions"
          placeholder="选择Java版本"
          style="width: 140px; flex-shrink: 0;"
        />
        <n-button
          :type="running ? 'error' : 'success'"
          @click="toggleRun"
          :disabled="!canInteract"
        >
          <template #icon>
            <svg v-if="!running" viewBox="0 0 24 24" width="16" height="16" style="vertical-align: middle;">
              <path d="M8 5v14l11-7z" fill="currentColor"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" width="16" height="16" style="vertical-align: middle;">
              <path d="M6 6h12v12H6z" fill="currentColor"/>
            </svg>
          </template>
          {{ running ? '停止' : '运行' }}
        </n-button>
      </n-space>
    </template>

    <div ref="terminalContainer" class="terminal-container" />
  </n-modal>
</template>

<script lang="ts" setup>
import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import {
  NModal, NButton, NSelect, NSpace,
  useMessage
} from 'naive-ui'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { DetectVenvs, RunPtyTool, StopPtySession } from '../../wailsjs/go/main/App'

interface ToolConfig {
  ID: string
  Name: string
  Type: string
  Path: string
  JavaVersion: string
  Description: string
  Category: string
}

interface JavaVersionEntry {
  version: string
  path: string
}

const props = defineProps<{
  tool: ToolConfig
  visible: boolean
  javaVersions: JavaVersionEntry[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
}>()

const message = useMessage()
const terminalContainer = ref<HTMLElement | null>(null)

const showModal = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v)
})

const running = ref(false)
const canInteract = ref(true)
const currentSessionID = ref('')
const selectedVenv = ref<string | null>(null)
const selectedJavaVersion = ref<string | null>(null)
const venvOptions = ref<{ label: string; value: string }[]>([])

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

const javaVersionSelectOptions = computed(() =>
  props.javaVersions.map(jv => ({ label: `Java ${jv.version}`, value: jv.version }))
)

function createTerminal(wsRef: { current: WebSocket | null }) {
  if (term) {
    term.dispose()
    term = null
    fitAddon = null
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: "'Cascadia Code', 'Consolas', 'Courier New', monospace",
    theme: { background: '#1e1e1e' },
    allowProposedApi: true,
    scrollback: 2000,
    cols: 120,
    rows: 35,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  if (terminalContainer.value) {
    term.open(terminalContainer.value)
    fitAndSyncSize(wsRef)

    resizeObserver = new ResizeObserver(() => {
      fitAndSyncSize(wsRef)
    })
    resizeObserver.observe(terminalContainer.value)

    requestAnimationFrame(() => fitAndSyncSize(wsRef))
    setTimeout(() => fitAndSyncSize(wsRef), 120)
  }

  term.onData((data) => {
    const sock = wsRef.current
    if (sock && sock.readyState === WebSocket.OPEN) {
      sock.send(data)
    }
  })

  term.onResize(({ cols, rows }) => {
    sendTerminalSize(wsRef, cols, rows)
  })
}

function fitAndSyncSize(wsRef: { current: WebSocket | null }) {
  if (!term || !fitAddon) return
  fitAddon.fit()
  sendTerminalSize(wsRef, term.cols, term.rows)
}

function sendTerminalSize(wsRef: { current: WebSocket | null }, cols?: number, rows?: number) {
  if (!term) return
  const sock = wsRef.current
  if (!sock || sock.readyState !== WebSocket.OPEN) return

  const nextCols = Math.max(2, Math.floor(cols ?? term.cols))
  const nextRows = Math.max(1, Math.floor(rows ?? term.rows))
  sock.send(JSON.stringify({ type: 'resize', cols: nextCols, rows: nextRows }))
}

function connectWebSocket(wsPort: number, sessionID: string, wsRef: { current: WebSocket | null }): Promise<void> {
  return new Promise((resolve, reject) => {
    const url = `ws://127.0.0.1:${wsPort}/ws?sid=${sessionID}`
    console.log('[CliTerminal] 连接 WebSocket:', url)

    const sock = new WebSocket(url)
    wsRef.current = sock

    sock.onopen = () => {
      console.log('[CliTerminal] WebSocket 已连接')
      fitAndSyncSize(wsRef)
      if (term) term.focus()
      resolve()
    }

    sock.onmessage = (e) => {
      if (typeof e.data === 'string') {
        if (e.data.startsWith('{')) {
          try {
            const msg = JSON.parse(e.data)
            if (msg.type === 'exit') {
              console.log('[CliTerminal] 进程已退出, code:', msg.code)
              onSessionExit(wsRef)
            }
          } catch { /* pass */ }
        } else {
          term?.write(e.data)
        }
      }
    }

    sock.onerror = (e) => {
      console.error('[CliTerminal] WebSocket 错误:', e)
      reject(new Error('WebSocket 连接失败'))
    }

    sock.onclose = () => {
      console.log('[CliTerminal] WebSocket 已关闭')
      onSessionExit(wsRef)
    }

    setTimeout(() => {
      if (sock.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket 连接超时'))
      }
    }, 5000)
  })
}

function onSessionExit(wsRef: { current: WebSocket | null }) {
  const sock = wsRef.current
  if (sock) {
    sock.onclose = null
    sock.onerror = null
    sock.onmessage = null
    sock.close()
    wsRef.current = null
  }
  if (running.value) {
    running.value = false
    currentSessionID.value = ''
    if (term) {
      term.writeln('\r\n\x1b[33m进程已结束\x1b[0m')
    }
  }
}

function closeSocket(wsRef: { current: WebSocket | null }) {
  const sock = wsRef.current
  if (sock) {
    sock.onclose = null
    sock.onerror = null
    sock.onmessage = null
    sock.close()
    wsRef.current = null
  }
}

function toggleRun() {
  if (running.value) {
    stopRun()
  } else {
    startRun()
  }
}

// Tracks the current session's WebSocket for cleanup
let sessionWs: { current: WebSocket | null } | null = null

async function startRun() {
  if (running.value) return

  console.log('[CliTerminal] ========== 开始运行 ==========')
  console.log('[CliTerminal] 工具:', props.tool.Name, 'Type:', props.tool.Type)
  console.log('[CliTerminal] 路径:', props.tool.Path)

  // Close previous session if any
  if (sessionWs) {
    closeSocket(sessionWs)
    sessionWs = null
  }

  running.value = true
  canInteract.value = false

  const wsRef: { current: WebSocket | null } = { current: null }
  sessionWs = wsRef
  createTerminal(wsRef)

  try {
    if (props.tool.Type === 'python') {
      console.log('[CliTerminal] 虚拟环境:', selectedVenv.value || '(未选择)')
    }
    if (props.tool.Type === 'java-cli') {
      console.log('[CliTerminal] Java版本:', selectedJavaVersion.value || props.tool.JavaVersion || '(默认)')
    }

    const result = await RunPtyTool(
      props.tool.ID,
      selectedJavaVersion.value || '',
      selectedVenv.value || ''
    )

    console.log('[CliTerminal] PTY 会话:', result)

    currentSessionID.value = result.sessionID as string
    const port = result.wsPort as number

    await connectWebSocket(port, result.sessionID as string, wsRef)
    canInteract.value = true
  } catch (e: any) {
    console.error('[CliTerminal] 启动失败:', e?.message || String(e))
    message.error(e?.message || String(e))
    running.value = false
    canInteract.value = true
    closeSocket(wsRef)
    sessionWs = null
  }
}

async function stopRun() {
  if (!currentSessionID.value) {
    console.log('[CliTerminal] stopRun: 无活跃会话')
    return
  }

  console.log('[CliTerminal] 停止会话:', currentSessionID.value)
  if (sessionWs) {
    closeSocket(sessionWs)
    sessionWs = null
  }

  try {
    await StopPtySession(currentSessionID.value)
    console.log('[CliTerminal] 会话已终止')
  } catch (e: any) {
    console.error('[CliTerminal] 停止失败:', e?.message || String(e))
  }

  running.value = false
  currentSessionID.value = ''
  canInteract.value = true
}

function handleAfterLeave() {
  console.log('[CliTerminal] 关闭终端, running:', running.value)
  if (running.value) {
    stopRun()
  }
  selectedVenv.value = null
  selectedJavaVersion.value = null
}

async function detectVenvs() {
  try {
    const dir = props.tool.Path.replace(/[\\/][^\\/]+$/, '') || '.'
    console.log('[CliTerminal] 检测虚拟环境，目录:', dir)
    const venvs = await DetectVenvs(dir)
    console.log('[CliTerminal] 发现虚拟环境:', venvs.length, '个', venvs)
    venvOptions.value = venvs.map(v => ({
      label: v.split(/[\\/]/).pop() || v,
      value: v
    }))
    selectedVenv.value = venvOptions.value[0]?.value ?? null
  } catch (e: any) {
    console.error('[CliTerminal] 检测虚拟环境失败:', e)
    venvOptions.value = []
  }
}

watch(() => props.visible, async (v) => {
  console.log('[CliTerminal] visible 变化:', v, '工具:', props.tool?.Name)
  if (v) {
    if (props.tool.Type === 'python') {
      detectVenvs()
    }
    if (props.tool.Type === 'java-cli' && !selectedJavaVersion.value) {
      selectedJavaVersion.value = props.tool.JavaVersion || ''
    }
    await nextTick()
  }
}, { immediate: true })

onBeforeUnmount(() => {
  if (sessionWs) {
    closeSocket(sessionWs)
    sessionWs = null
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  if (term) {
    term.dispose()
    term = null
    fitAddon = null
  }
})
</script>

<style scoped>
.terminal-container {
  height: 65vh;
  min-height: 400px;
  padding: 4px;
  background: #1e1e1e;
  text-align: left;
}

:deep(.n-card__content) {
  padding: 0 !important;
}
</style>
