import { clearSession, getToken } from './auth'

const API_BASE = '/bugs/api/v1'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (!(init.body instanceof FormData) && !headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json')
  }
  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    clearSession()
    if (!window.location.pathname.includes('/login')) {
      window.location.assign('/bugs/login')
    }
    throw new ApiError(401, 'unauthorized')
  }
  const text = await res.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      throw new ApiError(res.status, res.ok ? 'invalid server response' : text.trim() || res.statusText)
    }
  }
  if (!res.ok) {
    const errMsg =
      data && typeof data === 'object' && data !== null && 'error' in data
        ? String((data as { error: unknown }).error)
        : res.statusText
    throw new ApiError(res.status, errMsg)
  }
  return data as T
}

export type BugReport = {
  id: number
  url: string
  description: string
  page_parameters: string
  tested_with_user: string
  status: string
  priority: string
  cycle: number
  rejection_reason: string | null
  created_at: string
  updated_at: string
  reporter_user_id: number
  reporter_username: string
  fixer_user_id: number | null
  fixer_username: string
  last_history_status?: string
  last_history_actor?: string
}

export type BugAttachment = {
  id: number
  original_filename: string
  content_type: string
  size_bytes: number
  created_at: string
}

export type TeamMessage = {
  id: number
  body: string
  created_at: string
  author_user_id: number
  author_username: string
  author_role: string
}

export type UserProfile = {
  id: number
  username: string
  role: string
  first_name?: string
  last_name?: string
  gender?: string
  avatar_style?: string
  has_avatar?: boolean
  avatar_url?: string
  created_at?: string
}

export type PeriodStats = {
  bugs: number
  fixed_by_dev: number
  finished: number
}

export type BugStatsPayload = {
  total: PeriodStats & { pending_to_fix: number }
  today: PeriodStats
  yesterday: PeriodStats
  this_week: PeriodStats
  last_week: PeriodStats
  by_page?: Array<{ page: string; bugs: number }>
  username?: string
}

export const api = {
  login: (username: string, password: string) =>
    request<{
      token: string
      user: UserProfile
    }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<UserProfile>('/auth/me'),
  updateMe: (body: { gender?: string; avatar_style?: string }) =>
    request<UserProfile>('/auth/me', { method: 'PATCH', body: JSON.stringify(body) }),
  uploadAvatar: (file: File) => {
    const form = new FormData()
    form.append('avatar', file)
    return request<UserProfile>('/auth/me/avatar', { method: 'POST', body: form })
  },
  user: (username: string) => request<UserProfile>(`/users/${encodeURIComponent(username)}`),
  userAvatarUrl: (userId: number) => `${API_BASE}/users/${userId}/avatar`,
  users: () => request<UserProfile[]>('/users'),
  bugReports: () => request<BugReport[]>('/bug-reports'),
  bugReport: (id: number) => request<BugReport>(`/bug-reports/${id}`),
  createBug: (form: FormData) =>
    request<{ id: number; status: string }>('/bug-reports', { method: 'POST', body: form }),
  updateBugFields: (id: number, body: { description?: string; page_parameters?: string }) =>
    request<{ id: number; ok: boolean }>(`/bug-reports/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(body),
    }),
  updateStatus: (id: number, status: string, rejection_reason?: string) =>
    request<{ id: number; status: string }>(`/bug-reports/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status, rejection_reason }),
    }),
  deleteBug: (id: number) =>
    request<{ ok: boolean; id: number }>(`/bug-reports/${id}`, { method: 'DELETE' }),
  history: (id: number) =>
    request<
      Array<{
        id: number
        bug_report_id: number
        url: string
        status: string
        rejection_reason: string | null
        created_at: string
        actor_user_id: number
        actor_username: string
        reporter_username: string
      }>
    >(`/bug-reports/${id}/history`),
  attachments: (id: number) => request<BugAttachment[]>(`/bug-reports/${id}/attachments`),
  attachmentDownloadUrl: (bugId: number, attachmentId: number) =>
    `${API_BASE}/bug-reports/${bugId}/attachments/${attachmentId}`,
  notifications: () =>
    request<
      Array<{
        id: number
        bug_report_id: number
        kind: string
        is_read: boolean
        created_at: string
        message: string
        url: string
        bug_status: string
        fixer_username: string
        reporter_username: string
        rejection_reason: string | null
      }>
    >('/notifications'),
  unreadCount: () => request<{ count: number }>('/notifications/unread-count'),
  bugStats: () => request<BugStatsPayload>('/stats'),
  myBugStats: () => request<BugStatsPayload>('/stats/me'),
  userBugStats: (username: string) =>
    request<BugStatsPayload>(`/stats/user/${encodeURIComponent(username)}`),
  markRead: (id: number) => request<{ ok: boolean }>(`/notifications/${id}/read`, { method: 'PATCH' }),
  markAllRead: () => request<{ ok: boolean }>('/notifications/read-all', { method: 'POST' }),
  vapidPublicKey: () => request<{ publicKey: string }>('/push/vapid-public-key'),
  pushSubscribe: (body: { endpoint: string; keys: { p256dh: string; auth: string } }) =>
    request<{ ok: boolean }>('/push/subscribe', { method: 'POST', body: JSON.stringify(body) }),
  pushUnsubscribe: (endpoint: string) =>
    request<{ ok: boolean }>('/push/subscribe', {
      method: 'DELETE',
      body: JSON.stringify({ endpoint }),
    }),
  teamChat: (bugId: number) => request<TeamMessage[]>(`/bug-reports/${bugId}/team-chat`),
  postTeamChat: (bugId: number, body: string) =>
    request<TeamMessage>(`/bug-reports/${bugId}/team-chat`, {
      method: 'POST',
      body: JSON.stringify({ body }),
    }),
}
