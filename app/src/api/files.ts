import { get } from './base'

export type FileNodeDto = {
  key: string
  label: string
  path: string
  leaf: boolean
  size?: string
  icon: string
  children?: FileNodeDto[]
}

export function listNodeFiles(nodeId: string, dirPath: string) {
  const params = new URLSearchParams({ path: dirPath })
  return get<FileNodeDto[]>(`/api/nodes/${nodeId}/files?${params.toString()}`)
}
