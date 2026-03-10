import { get, post } from './base'

export type SessionDto = {
  id: string
  nodeId: string
  name: string
  process: string
  workspace: string
  status: string
  createdAt: string
}

export type SessionInput = {
  name: string
  process: string
  workspace: string
}

export function listNodeSessions(nodeId: string) {
  return get<SessionDto[]>(`/api/nodes/${nodeId}/sessions`)
}

export function updateSession(sessionId: string, payload: SessionInput) {
  return post<SessionDto>(`/api/sessions/${sessionId}/update`, payload)
}

export function deleteSession(sessionId: string) {
  return post<Record<string, never>>(`/api/sessions/${sessionId}/delete`)
}
