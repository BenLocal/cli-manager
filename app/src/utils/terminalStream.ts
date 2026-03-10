type TerminalOutputListener = (chunk: string) => void

const listeners = new Map<string, Set<TerminalOutputListener>>()

export function emitTerminalOutput(sessionId: string, chunk: string) {
  const sessionListeners = listeners.get(sessionId)
  if (!sessionListeners || !chunk) return

  for (const listener of sessionListeners) {
    listener(chunk)
  }
}

export function subscribeTerminalOutput(sessionId: string, listener: TerminalOutputListener) {
  const sessionListeners = listeners.get(sessionId) ?? new Set<TerminalOutputListener>()
  sessionListeners.add(listener)
  listeners.set(sessionId, sessionListeners)

  return () => {
    const current = listeners.get(sessionId)
    if (!current) return
    current.delete(listener)
    if (current.size === 0) {
      listeners.delete(sessionId)
    }
  }
}
