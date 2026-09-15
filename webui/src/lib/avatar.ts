/** Name-based avatar gender fallback. Explicit avatar_style wins in UI. */

const FEMALE_TOKENS = new Set(['sam', 'jordan', 'alexa', 'maria'])

export const AVATAR_STYLES = ['man', 'woman', 'nerd', 'robot', 'fox', 'abstract'] as const
export type AvatarStyle = (typeof AVATAR_STYLES)[number] | ''

export function normalizePersonKey(name: string): string {
  return name.toLowerCase().replace(/[._-]+/g, ' ').replace(/\s+/g, ' ').trim()
}

export function avatarGenderForName(name: string | null | undefined): 'man' | 'woman' | '' {
  const parts = normalizePersonKey(name || '').split(' ').filter(Boolean)
  if (parts.length === 0) return ''
  if (parts.some((p) => FEMALE_TOKENS.has(p))) return 'woman'
  return 'man'
}

export function hashName(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

export function resolveAvatarKind(
  style: string | null | undefined,
  gender: string | null | undefined,
  name: string | null | undefined,
): AvatarStyle | 'empty' {
  const s = (style || '').toLowerCase()
  if (s === 'man' || s === 'woman' || s === 'nerd' || s === 'robot' || s === 'fox' || s === 'abstract') {
    return s
  }
  const g = (gender || '').toLowerCase()
  if (g === 'man' || g === 'male') return 'man'
  if (g === 'woman' || g === 'female') return 'woman'
  const fromName = avatarGenderForName(name)
  if (fromName) return fromName
  return 'empty'
}
