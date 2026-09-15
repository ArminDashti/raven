export type Role = 'tester' | 'developer' | 'manager' | 'admin'

export type AuthUser = {
  id: number
  username: string
  role: Role
  first_name?: string
  last_name?: string
  gender?: string
  avatar_style?: string
  has_avatar?: boolean
}

const TOKEN_KEY = 'bug_report_token'
const USER_KEY = 'bug_report_user'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getUser(): AuthUser | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthUser
  } catch {
    return null
  }
}

export function setSession(token: string, user: AuthUser) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

export function isLoggedIn(): boolean {
  return !!getToken()
}
