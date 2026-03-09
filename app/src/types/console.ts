import type { TreeNode } from 'primevue/treenode'

export type NodeStatus = 'online' | 'warning' | 'offline'
export type ViewMode = 'terminal' | 'sessions'
export type ThemePresetKey = 'aura' | 'lara' | 'nora' | 'material'
export type SessionStatus = 'connecting' | 'live' | 'closed'

export interface NodeItem {
  id: string
  name: string
  status: NodeStatus
  ip: string
  port: string
  cpu: string
  memory: string
  type: string
  user: string
  password: string
  defaultProcess: string
  defaultWorkspace: string
}

export interface SessionItem {
  id: string
  name: string
  workspace: string
  history: string[]
  files: TreeNode[]
  createdAt: string
  status: SessionStatus
}

export interface NodeSessionState {
  activeSessionId: string
  sessions: SessionItem[]
}

export type NodeSessionRecord = Record<string, NodeSessionState>

export interface ConsoleStats {
  online: number
  warning: number
  sessions: number
}

export interface NewNodePayload {
  name: string
  ip: string
  port: string
  user: string
  password: string
}

export interface SessionCreatePayload {
  process: string
  workspace: string
}

export type SessionDialogMode = 'create' | 'edit'

export interface TerminalSessionPayload {
  sessionId: string
  nodeId: string
  process: string
  workspace: string
  createdAt: string
}

export interface TerminalOutputPayload {
  sessionId: string
  nodeId: string
  data: string
}

export interface TerminalClosePayload {
  sessionId: string
  nodeId: string
  reason: string
}

export interface TerminalErrorPayload {
  sessionId?: string
  nodeId?: string
  message: string
}

export interface ThemeOption {
  label: string
  value: ThemePresetKey
}
