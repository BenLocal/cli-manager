import { get, post } from './base'

export type NodeDto = {
  id: string
  name: string
  ip: string
  port: string
  user: string
  password: string
  status: string
  cpu: string
  memory: string
  type: string
  defaultProcess: string
  defaultWorkspace: string
}

export type NodeInput = {
  name: string
  ip: string
  port: string
  user: string
  password: string
}

export function listNodes() {
  return get<NodeDto[]>('/api/nodes')
}

export function createNode(payload: NodeInput) {
  return post<NodeDto>('/api/nodes', payload)
}

export function updateNode(nodeId: string, payload: NodeInput) {
  return post<NodeDto>(`/api/nodes/${nodeId}/update`, payload)
}

export function deleteNode(nodeId: string) {
  return post<Record<string, never>>(`/api/nodes/${nodeId}/delete`)
}
