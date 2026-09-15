import { toJalaali } from 'jalaali-js'

export function toShamsiDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const j = toJalaali(d.getFullYear(), d.getMonth() + 1, d.getDate())
  const yyyy = String(j.jy)
  const mo = String(j.jm).padStart(2, '0')
  const day = String(j.jd).padStart(2, '0')
  return `${yyyy}-${mo}-${day}`
}

function relativeDaysAgo(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const ms = Date.now() - d.getTime()
  const days = Math.max(0, Math.floor(ms / (24 * 60 * 60 * 1000)))
  if (days === 0) return 'today'
  if (days === 1) return '1 day ago'
  return `${days} days ago`
}

export function toShamsiDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const j = toJalaali(d.getFullYear(), d.getMonth() + 1, d.getDate())
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const yyyy = String(j.jy)
  const mo = String(j.jm).padStart(2, '0')
  const day = String(j.jd).padStart(2, '0')
  const ago = relativeDaysAgo(iso)
  return ago ? `${yyyy}-${mo}-${day} ${hh}:${mm} (${ago})` : `${yyyy}-${mo}-${day} ${hh}:${mm}`
}

export function autoDir(value: string): 'rtl' | 'ltr' {
  return /[\u0600-\u06FF]/.test(value || '') ? 'rtl' : 'ltr'
}

/** Grid label for a bug URL: `.aspx` page name without extension; otherwise full URL as-is. */
export function displayBugUrl(url: string): string {
  const raw = (url || '').trim()
  if (!raw) return raw
  try {
    const parsed = new URL(raw)
    const match = parsed.pathname.match(/\/([^/]+)\.aspx$/i)
    if (match) return match[1]
    return raw
  } catch {
    const match = raw.match(/(?:^|\/)([^/?#]+)\.aspx(?:[?#]|$)/i)
    if (match) return match[1]
    return raw
  }
}

/**
 * Rewrite a bug URL's origin using a personal base override (path/query/hash kept).
 * Empty override ⇒ original URL. Invalid base ⇒ original URL.
 */
export function applyBugBaseUrl(url: string, baseOverride: string): string {
  const raw = (url || '').trim()
  const base = (baseOverride || '').trim().replace(/\/+$/, '')
  if (!raw || !base) return raw
  try {
    const parsed = new URL(raw)
    const suffix = `${parsed.pathname}${parsed.search}${parsed.hash}`
    const joined = base.endsWith('/') ? base.slice(0, -1) + suffix : base + suffix
    // Validate result is absolute
    void new URL(joined)
    return joined
  } catch {
    return raw
  }
}
