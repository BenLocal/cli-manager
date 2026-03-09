import {
  computed,
  onMounted,
  onUnmounted,
  reactive,
  ref,
  shallowRef,
} from 'vue'
import {
  HubConnectionBuilder,
  HubConnectionState,
  LogLevel,
  type HubConnection,
} from '@microsoft/signalr'

import { createNodeSessions, createWorkspaceTree, initialNodes } from '../data/console'
import type {
  NewNodePayload,
  NodeItem,
  NodeSessionRecord,
  NodeSessionState,
  SessionDialogMode,
  SessionItem,
  SessionCreatePayload,
  TerminalClosePayload,
  TerminalErrorPayload,
  TerminalOutputPayload,
  TerminalSessionPayload,
  ViewMode,
} from '../types/console'

export function useConsoleManager() {
  const nodes = ref<NodeItem[]>(structuredClone(initialNodes))
  const nodeSessions = reactive<NodeSessionRecord>(createNodeSessions())
  const selectedNodeId = ref(nodes.value[0]?.id ?? '')
  const showSidebar = ref(true)
  const showWorkspace = ref(true)
  const showSessionDialog = ref(false)
  const sessionDialogMode = ref<SessionDialogMode>('create')
  const editingSessionId = ref('')
  const viewMode = ref<ViewMode>('terminal')
  const hubConnection = shallowRef<HubConnection | null>(null)
  let connectPromise: Promise<void> | null = null

  const selectedNode = computed(() => nodes.value.find((node) => node.id === selectedNodeId.value) ?? null)

  const selectedNodeState = computed(() => {
    const node = selectedNode.value
    if (!node) return null
    return nodeSessions[node.id] ?? null
  })

  const activeSession = computed(() => {
    const nodeState = selectedNodeState.value
    if (!nodeState) return null
    return nodeState.sessions.find((session) => session.id === nodeState.activeSessionId) ?? null
  })

  const overviewStats = computed(() => ({
    online: nodes.value.filter((node) => node.status === 'online').length,
    warning: nodes.value.filter((node) => node.status === 'warning').length,
    sessions: Object.values(nodeSessions).reduce(
      (count, state) => count + state.sessions.filter((session) => session.status === 'live').length,
      0,
    ),
  }))

  const sessionDialogDefaults = computed(() => {
    if (sessionDialogMode.value === 'edit' && selectedNode.value && editingSessionId.value) {
      const session = nodeSessions[selectedNode.value.id]?.sessions.find((item) => item.id === editingSessionId.value)
      if (session) {
        return {
          process: session.name,
          workspace: session.workspace,
        }
      }
    }

    return {
      process: selectedNode.value?.defaultProcess ?? 'bash',
      workspace: selectedNode.value?.defaultWorkspace ?? '/root',
    }
  })

  onMounted(() => {
    void ensureHubConnection()
  })

  onUnmounted(() => {
    const connection = hubConnection.value
    hubConnection.value = null
    if (connection) {
      void connection.stop()
    }
  })

  function ensureNodeState(nodeId: string): NodeSessionState {
    if (!nodeSessions[nodeId]) {
      nodeSessions[nodeId] = {
        activeSessionId: '',
        sessions: [],
      }
    }
    return nodeSessions[nodeId]
  }

  function selectNode(nodeId: string) {
    selectedNodeId.value = nodeId
    viewMode.value = 'terminal'
  }

  function openSessionDialog(sessionId?: string) {
    if (!selectedNode.value) return
    sessionDialogMode.value = sessionId ? 'edit' : 'create'
    editingSessionId.value = sessionId ?? ''
    showSessionDialog.value = true
  }

  async function saveSession(payload: SessionCreatePayload) {
    if (!selectedNode.value) return
    if (sessionDialogMode.value === 'edit') {
      const session = nodeSessions[selectedNode.value.id]?.sessions.find((item) => item.id === editingSessionId.value)
      if (!session) return
      session.name = payload.process
      session.workspace = payload.workspace
      session.files = createWorkspaceTree(payload.workspace)
      showSessionDialog.value = false
      editingSessionId.value = ''
      return
    }

    const nodeId = selectedNode.value.id
    const nodeState = ensureNodeState(nodeId)
    const pendingId = `pending-${Date.now()}`
    nodeState.sessions.push({
      id: pendingId,
      name: payload.process,
      workspace: payload.workspace,
      createdAt: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
      history: [
        `[SYSTEM] 正在为节点 ${selectedNode.value.name} 创建会话`,
        `[SYSTEM] 工作目录准备切换至: ${payload.workspace}`,
        `[EXEC] 启动进程请求已提交: ${payload.process}`,
      ],
      files: createWorkspaceTree(payload.workspace),
      status: 'connecting',
    })
    nodeState.activeSessionId = pendingId
    showSessionDialog.value = false
    editingSessionId.value = ''
    viewMode.value = 'terminal'

    try {
      const connection = await ensureHubConnection()
      await connection.invoke('CreateSession', nodeId, payload.process, payload.workspace)
    } catch (error) {
      const pending = nodeState.sessions.find((item) => item.id === pendingId)
      if (pending) {
        pending.status = 'closed'
        pending.history.push('[SYSTEM] 会话创建失败，已切换为本地占位状态')
      }
      console.error('create session failed', error)
    }
  }

  function switchSession(sessionId: string) {
    if (!selectedNode.value) return
    const nodeState = nodeSessions[selectedNode.value.id]
    if (!nodeState) return
    nodeState.activeSessionId = sessionId
    viewMode.value = 'terminal'
  }

  async function deleteSession(sessionId: string) {
    if (!selectedNode.value) return
    const nodeState = nodeSessions[selectedNode.value.id]
    if (!nodeState) return

    const index = nodeState.sessions.findIndex((session) => session.id === sessionId)
    if (index === -1) return

    const session = nodeState.sessions[index]
    if (!session) return

    try {
      if (session.status !== 'closed') {
        const connection = await ensureHubConnection()
        await connection.invoke('CloseSession', sessionId)
      }
    } catch (error) {
      console.error('close session failed', error)
    }

    removeSession(nodeState, sessionId, index)
  }

  function addNode(payload: NewNodePayload) {
    const nodeId = `node-${Date.now()}`
    const node: NodeItem = {
      id: nodeId,
      name: payload.name,
      ip: payload.ip,
      port: payload.port,
      user: payload.user,
      password: payload.password,
      status: 'online',
      cpu: '0%',
      memory: '0GB',
      type: 'Worker',
      defaultProcess: 'bash',
      defaultWorkspace: '/root',
    }

    nodes.value.unshift(node)
    nodeSessions[nodeId] = {
      activeSessionId: '',
      sessions: [],
    }
    selectNode(nodeId)
  }

  function updateNode(nodeId: string, payload: NewNodePayload) {
    const node = nodes.value.find((item) => item.id === nodeId)
    if (!node) return
    node.name = payload.name
    node.ip = payload.ip
    node.port = payload.port
    node.user = payload.user
    node.password = payload.password
  }

  function deleteNode(nodeId: string) {
    const index = nodes.value.findIndex((item) => item.id === nodeId)
    if (index === -1) return
    nodes.value.splice(index, 1)
    delete nodeSessions[nodeId]

    if (selectedNodeId.value === nodeId) {
      selectedNodeId.value = nodes.value[0]?.id ?? ''
    }
  }

  async function submitCommand(command: string) {
    const session = activeSession.value
    if (!session || session.status !== 'live') return

    try {
      const connection = await ensureHubConnection()
      await connection.invoke('Input', session.id, `${command}\n`)
    } catch (error) {
      console.error('submit command failed', error)
    }
  }

  async function ensureHubConnection() {
    if (hubConnection.value && hubConnection.value.state === HubConnectionState.Connected) {
      return hubConnection.value
    }

    if (connectPromise) {
      await connectPromise
      if (!hubConnection.value) throw new Error('signalr connection unavailable')
      return hubConnection.value
    }

    const connection = new HubConnectionBuilder()
      .withUrl('/hub/terminal')
      .withAutomaticReconnect()
      .configureLogging(LogLevel.Warning)
      .build()

    registerHubCallbacks(connection)

    connectPromise = connection
      .start()
      .then(() => {
        hubConnection.value = connection
      })
      .finally(() => {
        connectPromise = null
      })

    await connectPromise
    if (!hubConnection.value) throw new Error('signalr connection unavailable')
    return hubConnection.value
  }

  function registerHubCallbacks(connection: HubConnection) {
    connection.on('SessionCreated', (payload: TerminalSessionPayload) => {
      const nodeState = ensureNodeState(payload.nodeId)
      const existing =
        nodeState.sessions.find((session) => session.id === payload.sessionId) ??
        nodeState.sessions.find(
          (session) =>
            session.status === 'connecting' &&
            session.name === payload.process &&
            session.workspace === payload.workspace,
        )
      if (existing) {
        existing.id = payload.sessionId
        existing.name = payload.process
        existing.workspace = payload.workspace
        existing.createdAt = payload.createdAt
        existing.files = createWorkspaceTree(payload.workspace)
        existing.history = [
          `[SYSTEM] 成功连接至会话 ${payload.sessionId}`,
          `[SYSTEM] 工作目录已切换至: ${payload.workspace}`,
          `[EXEC] 初始化进程已启动: ${payload.process}`,
        ]
        existing.status = 'live'
      } else {
        nodeState.sessions.push({
          id: payload.sessionId,
          name: payload.process,
          workspace: payload.workspace,
          createdAt: payload.createdAt,
          history: [
            `[SYSTEM] SignalR connected at ${payload.createdAt}`,
            `[SYSTEM] Workspace set to ${payload.workspace}`,
            `[EXEC] process launched: ${payload.process}`,
          ],
          files: createWorkspaceTree(payload.workspace),
          status: 'live',
        })
      }
      nodeState.activeSessionId = payload.sessionId
      selectedNodeId.value = payload.nodeId
      viewMode.value = 'terminal'
    })

    connection.on('SessionOutput', (payload: TerminalOutputPayload) => {
      const session = findSession(payload.nodeId, payload.sessionId)
      if (!session) return
      appendOutput(session, payload.data)
      session.status = 'live'
    })

    connection.on('SessionClosed', (payload: TerminalClosePayload) => {
      const session = findSession(payload.nodeId, payload.sessionId)
      if (!session) return
      session.status = 'closed'
      if (payload.reason) {
        session.history.push(`[SYSTEM] ${payload.reason}`)
      }
    })

    connection.on('SessionError', (payload: TerminalErrorPayload) => {
      if (payload.sessionId && payload.nodeId) {
        const session = findSession(payload.nodeId, payload.sessionId)
        if (session) {
          session.history.push(`[SYSTEM] ${payload.message}`)
          return
        }
      }
      console.error('session error', payload.message)
    })
  }

  function findSession(nodeId: string, sessionId: string) {
    const nodeState = nodeSessions[nodeId]
    return nodeState?.sessions.find((session) => session.id === sessionId) ?? null
  }

  function removeSession(nodeState: NodeSessionState, sessionId: string, knownIndex?: number) {
    const index = knownIndex ?? nodeState.sessions.findIndex((session) => session.id === sessionId)
    if (index === -1) return
    nodeState.sessions.splice(index, 1)

    if (nodeState.activeSessionId === sessionId) {
      const fallback = nodeState.sessions[Math.max(index - 1, 0)] ?? nodeState.sessions[0]
      nodeState.activeSessionId = fallback?.id ?? ''
    }
  }

  function createDefaultNodeState(node: NodeItem): NodeSessionState {
    const sessionId = `sess-${node.id}-1`
    return {
      activeSessionId: sessionId,
      sessions: [
        {
          id: sessionId,
          name: node.defaultProcess,
          workspace: node.defaultWorkspace,
          createdAt: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
          history: [
            `[SYSTEM] 成功连接至节点 ${node.name} (${node.ip})`,
            `[SYSTEM] 工作目录已切换至: ${node.defaultWorkspace}`,
            `[EXEC] 正在启动初始化进程: ${node.defaultProcess}...`,
          ],
          files: createWorkspaceTree(node.defaultWorkspace),
          status: 'live',
        },
      ],
    }
  }

  function appendOutput(session: SessionItem, chunk: string) {
    const lines = chunk.replace(/\r/g, '').split('\n')
    for (const line of lines) {
      if (!line) continue
      session.history.push(line)
    }

    if (session.history.length > 400) {
      session.history.splice(0, session.history.length - 400)
    }
  }

  return {
    nodes,
    nodeSessions,
    selectedNodeId,
    selectedNode,
    selectedNodeState,
    activeSession,
    showSidebar,
    showWorkspace,
    showSessionDialog,
    sessionDialogMode,
    sessionDialogDefaults,
    viewMode,
    overviewStats,
    selectNode,
    openSessionDialog,
    saveSession,
    switchSession,
    deleteSession,
    addNode,
    updateNode,
    deleteNode,
    submitCommand,
  }
}
