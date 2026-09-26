export interface ApiError {
  error: { code: string; message: string; request_id?: string }
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '/api/v1').replace(/\/$/, '')

export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  })

  if (!response.ok) {
    let payload: ApiError | undefined
    try { payload = await response.json() as ApiError } catch { /* non-JSON error */ }
    throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
  }

  if (response.status === 204) return undefined as T
  return await response.json() as T
}

export const healthApi = {
  get: () => apiRequest<{ status: string }>('/health'),
}
