type ApiSuccess<T> = {
  code: number
  message: string
  data: T
}

type ApiError = {
  code: number
  message: string
}

type ApiResponse<T> = ApiSuccess<T> | ApiError

async function request<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`)
  }

  const payload = (await response.json()) as ApiResponse<T>
  if ('data' in payload && payload.code === 200) {
    return payload.data
  }

  throw new Error(payload.message || 'request failed')
}

export function get<T>(input: RequestInfo | URL) {
  return request<T>(input)
}

export function post<T>(input: RequestInfo | URL, body?: unknown) {
  return request<T>(input, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
}
