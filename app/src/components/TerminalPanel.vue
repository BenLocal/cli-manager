<script setup lang="ts">
import Card from 'primevue/card'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from 'xterm'
import 'xterm/css/xterm.css'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

import type { SessionItem } from '../types/console'

const props = defineProps<{
  session: SessionItem
  nodeName: string
  nodeUser: string
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
let renderedLineCount = 0

function colorize(line: string) {
  if (line.startsWith('[SYSTEM]')) return `\u001b[36m${line}\u001b[0m`
  if (line.startsWith('[EXEC]')) return `\u001b[32m${line}\u001b[0m`
  return line
}

function writeHistory(force = false) {
  if (!terminal) return

  const sessionChanged = renderedSessionId !== props.session.id
  const historyShrunk = props.session.history.length < renderedLineCount

  if (force || sessionChanged || historyShrunk) {
    terminal.reset()
    renderedSessionId = props.session.id
    renderedLineCount = 0
  }

  const nextLines = props.session.history.slice(renderedLineCount)
  if (!nextLines.length) return

  for (const line of nextLines) {
    terminal.writeln(colorize(line))
  }

  renderedLineCount = props.session.history.length
  terminal.scrollToBottom()
}

function fitTerminal() {
  nextTick(() => {
    fitAddon.fit()
    if (terminal) {
      emit('resize', { cols: terminal.cols, rows: terminal.rows })
    }
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
  terminal.loadAddon(fitAddon)
  terminal.open(terminalHost.value!)
  terminal.onData((data) => {
    if (props.session.status !== 'live') return
    emit('terminal-input', data)
  })
  writeHistory(true)
  fitTerminal()

  resizeObserver = new ResizeObserver(() => {
    fitTerminal()
  })

  if (terminalHost.value) {
    resizeObserver.observe(terminalHost.value)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  terminal?.dispose()
  terminal = null
})

watch(
  () => [props.session.id, props.session.history.length],
  () => {
    writeHistory()
    fitTerminal()
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
