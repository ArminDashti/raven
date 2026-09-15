import { getUser } from './auth'

const BASE_URL_PREFIX = 'raven_bug_base_url:'

function keyForUser(userId: number): string {
  return `${BASE_URL_PREFIX}${userId}`
}

/** Per-user (this browser) override for bug link origins. */
export function getBugBaseUrlOverride(): string {
  const user = getUser()
  if (!user || typeof localStorage === 'undefined') return ''
  return localStorage.getItem(keyForUser(user.id)) || ''
}

export function setBugBaseUrlOverride(value: string): void {
  const user = getUser()
  if (!user || typeof localStorage === 'undefined') return
  const trimmed = value.trim()
  const key = keyForUser(user.id)
  if (!trimmed) {
    localStorage.removeItem(key)
    return
  }
  localStorage.setItem(key, trimmed)
}
