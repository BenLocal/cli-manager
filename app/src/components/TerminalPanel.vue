<script setup lang="ts">
import Card from 'primevue/card'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from 'xterm'
import 'xterm/css/xterm.css'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import type { SessionItem } from '../types/console'
import { subscribeTerminalOutput } from '../utils/terminalStream'

const props = defineProps<{
  session: SessionItem
  nodeName: string
  nodeUser: string
  sidebarVisible?: boolean
  workspaceVisible?: boolean
}>()

const emit = defineEmits<{
  (event: 'terminal-input', value: string): void
  (event: 'resize', value: { cols: number; rows: number }): void
}>()

const terminalHost = ref<HTMLElement | null>(null)
const fitAddon = new FitAddon()
let terminal: Terminal | null = null
let resizeObserver: ResizeObserver | null = null
let renderedSessionId = ''
let renderedChunkCount = 0
let rafId = 0
let lastResizeKey = ''
let writeRafId = 0
let unsubscribeTerminalOutput: (() => void) | null = null

function focusTerminal() {
  terminal?.focus()
}

function isLocalMessage(chunk: string) {
  return chunk.startsWith('[SYSTEM]') || chunk.startsWith('[EXEC]')
}

function colorize(line: string) {
  if (line.startsWith('[SYSTEM]')) return `\u001b[36m${line}\u001b[0m\r\n`
  if (line.startsWith('[EXEC]')) return `\u001b[32m${line}\u001b[0m\r\n`
  return line
}

function writeHistory(force = false) {
  if (!terminal) return

  const sessionChanged = renderedSessionId !== props.session.id
  const historyShrunk = props.session.history.length < renderedChunkCount

  if (force || sessionChanged || historyShrunk) {
    terminal.reset()
    renderedSessionId = props.session.id
    renderedChunkCount = 0
    lastResizeKey = ''
  }

  const nextChunks = props.session.history.slice(renderedChunkCount)
  if (!nextChunks.length) return

  const output = nextChunks
    .filter((chunk) => force || isLocalMessage(chunk) || props.session.status !== 'live')
    .map((chunk) => (isLocalMessage(chunk) ? colorize(chunk) : chunk))
    .join('')

  if (output) {
    terminal.write(output)
  }

  renderedChunkCount = props.session.history.length
  terminal.scrollToBottom()
}

function scheduleWriteHistory(force = false) {
  if (writeRafId) {
    cancelAnimationFrame(writeRafId)
  }
  writeRafId = requestAnimationFrame(() => {
    writeRafId = 0
    writeHistory(force)
  })
}

function fitTerminal() {
  nextTick(() => {
    if (rafId) {
      cancelAnimationFrame(rafId)
    }
    rafId = requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        fitAddon.fit()
        if (terminal) {
          const resizeKey = `${terminal.cols}x${terminal.rows}`
          if (resizeKey === lastResizeKey) {
            return
          }
          lastResizeKey = resizeKey
          emit('resize', { cols: terminal.cols, rows: terminal.rows })
        }
      })
    })
  })
}

function bindTerminalStream() {
  unsubscribeTerminalOutput?.()
  unsubscribeTerminalOutput = subscribeTerminalOutput(props.session.id, (chunk) => {
    if (!terminal) return
    terminal.write(chunk)
    terminal.scrollToBottom()
    renderedChunkCount = props.session.history.length
  })
}

onMounted(() => {
  terminal = new Terminal({
    convertEol: true,
    cursorBlink: true,
    disableStdin: false,
    fontFamily: 'JetBrains Mono, Cascadia Code, Consolas, monospace',
    fontSize: 12,
    lineHeight: 1.55,
    theme: {
      background: '#020812',
      foreground: '#d9f4ff',
      cursor: '#08d2ff',
      black: '#020812',
      brightBlack: '#6f8097',
      cyan: '#05d7ff',
      green: '#19f3c6',
    },
  })
  ;(terminal as Terminal & {
    attachCustomWheelEventHandler?: (handler: (event: WheelEvent) => boolean) => void
  }).attachCustomWheelEventHandler?.((event: WheelEvent) => {
    const viewport = terminalHost.value?.querySelector('.xterm-viewport')
    if (!viewport) {
      return false
    }

    viewport.scrollTop += event.deltaY
    return false
  })
  terminal.loadAddon(fitAddon)
  terminal.open(terminalHost.value!)
  terminalHost.value?.addEventListener('click', focusTerminal)
  terminal.onData((data) => {
    if (props.session.status !== 'live') return
    emit('terminal-input', data)
  })
  scheduleWriteHistory(true)
  fitTerminal()
  focusTerminal()
  bindTerminalStream()

  resizeObserver = new ResizeObserver(() => {
    fitTerminal()
  })

  if (terminalHost.value) {
    resizeObserver.observe(terminalHost.value)
  }
  window.addEventListener('resize', fitTerminal)
})

onUnmounted(() => {
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = 0
  }
  if (writeRafId) {
    cancelAnimationFrame(writeRafId)
    writeRafId = 0
  }
  unsubscribeTerminalOutput?.()
  unsubscribeTerminalOutput = null
  terminalHost.value?.removeEventListener('click', focusTerminal)
  window.removeEventListener('resize', fitTerminal)
  resizeObserver?.disconnect()
  resizeObserver = null
  terminal?.dispose()
  terminal = null
})

watch(
  () => props.session.id,
  () => {
    bindTerminalStream()
    scheduleWriteHistory(true)
  },
)

watch(
  () => props.session.history.length,
  () => {
    scheduleWriteHistory()
  },
)

watch(
  () => [props.session.id, props.session.workspace, props.sidebarVisible, props.workspaceVisible],
  () => {
    scheduleWriteHistory()
    fitTerminal()
    focusTerminal()
  },
)
</script>

<template>
  <Card class="terminal-card">
    <template #title>
      <div class="section-title section-title--terminal">
        <span>TTY: {{ session.name.toUpperCase() }}</span>
        <span>DIR: {{ session.workspace.toUpperCase() }}</span>
      </div>
    </template>

    <template #content>
      <div class="terminal-card__scroll">
        <div ref="terminalHost" class="terminal-xterm"></div>
      </div>
    </template>
  </Card>
</template>
