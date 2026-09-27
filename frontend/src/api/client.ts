export interface ApiError {
  error: { code: string; message: string; request_id?: string }
}

export interface Project {
  id: string
  user_id: string
  name: string
  description?: string
  status: 'active' | 'archived' | 'deleted'
  created_at: string
  updated_at: string
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? '/api/v1').replace(/\/$/, '')

export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const token = localStorage.getItem('web-studio-access-token')
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init.headers,
    },
  })

  if (!response.ok) {
    let payload: ApiError | undefined
    try {
      payload = await response.json() as ApiError
    } catch {
      // non-JSON error
    }
    throw new Error(payload?.error.message ?? `Request failed with status ${response.status}`)
  }

  if (response.status === 204) return undefined as T
  return await response.json() as T
}

export const healthApi = {
  get: () => apiRequest<{ status: string }>('/health'),
}

export const projectsApi = {
  list: () => apiRequest<{ projects: Project[] }>('/projects'),
  get: (id: string) => apiRequest<Project>(`/projects/${id}`),
  create: (input: { name: string; description?: string }) =>
    apiRequest<Project>('/projects', { method: 'POST', body: JSON.stringify(input) }),
  update: (id: string, input: { name: string; description?: string; status: Project['status'] }) =>
    apiRequest<Project>(`/projects/${id}`, { method: 'PATCH', body: JSON.stringify(input) }),
  archive: (id: string) =>
    apiRequest<Project>(`/projects/${id}`, { method: 'DELETE' }),
}

export type IterationType =
  | 'idea'
  | 'sketch'
  | 'generation'
  | 'selection'
  | 'composition'
  | 'prompt'
  | 'manual_edit'
  | 'final'

export interface Iteration {
  id: string
  project_id: string
  parent_iteration_id?: string
  type: IterationType
  title?: string
  description?: string
  created_at: string
}

export const iterationsApi = {
  list: (projectId: string) =>
    apiRequest<{ iterations: Iteration[] }>(`/projects/${projectId}/iterations`),
  create: (
    projectId: string,
    input: {
      parent_iteration_id?: string
      type: IterationType
      title?: string
      description?: string
    },
  ) =>
    apiRequest<Iteration>(`/projects/${projectId}/iterations`, {
      method: 'POST',
      body: JSON.stringify(input),
    }),
  get: (projectId: string, iterationId: string) =>
    apiRequest<Iteration>(`/projects/${projectId}/iterations/${iterationId}`),
  restore: (projectId: string, iterationId: string) =>
    apiRequest<Iteration>(`/projects/${projectId}/iterations/${iterationId}/restore`, {
      method: 'POST',
    }),
}


export type GenerationStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface Generation {
  id: string
  project_id: string
  iteration_id?: string
  user_id: string
  provider: string
  model: string
  model_version?: string
  prompt: string
  negative_prompt?: string
  seed?: number
  aspect_ratio?: string
  parameters?: Record<string, unknown>
  status: GenerationStatus
  provider_job_id?: string
  error_code?: string
  error_message?: string
  cost?: number
  created_at: string
  started_at?: string
  completed_at?: string
}

export const generationsApi = {
  create: (
    projectId: string,
    input: {
      iteration_id?: string
      prompt: string
      negative_prompt?: string
      seed?: number
      aspect_ratio?: string
      parameters?: Record<string, unknown>
    },
    idempotencyKey: string,
  ) =>
    apiRequest<Generation>(`/projects/${projectId}/generations`, {
      method: 'POST',
      headers: { 'Idempotency-Key': idempotencyKey },
      body: JSON.stringify(input),
    }),
  get: (projectId: string, generationId: string) =>
    apiRequest<Generation>(`/projects/${projectId}/generations/${generationId}`),
  cancel: (projectId: string, generationId: string) =>
    apiRequest<Generation>(`/projects/${projectId}/generations/${generationId}/cancel`, {
      method: 'POST',
    }),
}
