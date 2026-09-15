const STORAGE_KEY = 'raven-bug-theme'

export type ThemeMode = 'light' | 'dark'

export function getStoredTheme(): ThemeMode | null {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw === 'light' || raw === 'dark') return raw
  return null
}

export function resolveTheme(): ThemeMode {
  return getStoredTheme() ?? 'dark'
}

export function applyTheme(mode: ThemeMode): void {
  const root = document.documentElement
  root.classList.toggle('dark', mode === 'dark')
  root.style.colorScheme = mode
}

export function setTheme(mode: ThemeMode): void {
  localStorage.setItem(STORAGE_KEY, mode)
  applyTheme(mode)
}

export function toggleTheme(): ThemeMode {
  const next: ThemeMode = resolveTheme() === 'dark' ? 'light' : 'dark'
  setTheme(next)
  return next
}

export function initTheme(): ThemeMode {
  const mode = resolveTheme()
  applyTheme(mode)
  return mode
}
