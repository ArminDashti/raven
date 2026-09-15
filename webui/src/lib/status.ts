import { toShamsiDate } from './format'

/** Optional username → "First Last" map; set from /users so login usernames stay unchanged. */
let personNameLookup: Record<string, string> = {}

export function setPersonNameLookup(map: Record<string, string>) {
  personNameLookup = map
}

export function fullNameFromParts(
  firstName?: string | null,
  lastName?: string | null,
  username?: string | null,
): string {
  const full = [firstName, lastName].map((p) => (p || '').trim()).filter(Boolean).join(' ')
  if (full) return full
  return displayPersonName(username)
}

/** Title-case a username for status copy (armin → Armin), or resolved First Last when known. */
export function displayPersonName(name: string | null | undefined): string {
  const raw = (name || '').trim()
  if (!raw) return ''
  const looked = personNameLookup[raw.toLowerCase()]
  if (looked) return looked
  return raw
    .split(/[\s._-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}
export function joinNames(names: Array<string | null | undefined>): string {
  const cleaned = names.map(displayPersonName).filter(Boolean)
  if (cleaned.length === 0) return ''
  if (cleaned.length === 1) return cleaned[0]
  if (cleaned.length === 2) return `${cleaned[0]} or ${cleaned[1]}`
  return `${cleaned.slice(0, -1).join(', ')}, or ${cleaned[cleaned.length - 1]}`
}

export type StatusPeople = {
  fixerUsername?: string | null
  reporterUsername?: string | null
  actorUsername?: string | null
  developerNames?: string[]
  testerNames?: string[]
  managerNames?: string[]
  closedAt?: string | null
}

export type BadgeStatusSource = {
  status: string
  last_history_status?: string | null
  last_history_actor?: string | null
}

/** Prefer "team_message" badge when that is the latest history event. */
export function badgeStatus(row: BadgeStatusSource): string {
  if ((row.last_history_status || '').toLowerCase() === 'team_message') return 'team_message'
  return row.status
}

export function badgeActorUsername(row: BadgeStatusSource): string | undefined {
  if ((row.last_history_status || '').toLowerCase() === 'team_message') {
    return row.last_history_actor || undefined
  }
  return undefined
}

export function statusLabel(status: string, people: StatusPeople = {}): string {
  const fixer = displayPersonName(people.fixerUsername)
  const reporter = displayPersonName(people.reporterUsername)
  const actor = displayPersonName(people.actorUsername)
  const developers = joinNames(
    people.developerNames?.length ? people.developerNames : fixer ? [fixer] : [],
  )
  const managers = joinNames(people.managerNames || [])

  switch (status) {
    case 'reported':
    case 'pending':
      if (fixer) return `Waiting for ${fixer} to fix it`
      if (developers) return `Waiting for ${developers} to fix it`
      return 'Waiting for a fixer'
    case 'fixed':
    case 'changed_by_dev':
      // Always the bug's own reporter (its tester) — never every tester role.
      if (reporter) return `Waiting for ${reporter} to approve it`
      return 'Waiting for a tester to approve it'
    case 'waiting_manager': {
      const reporterRaw = (people.reporterUsername || '').trim().toLowerCase()
      const reporterIsManager =
        !!reporter &&
        (people.managerNames || []).some(
          (n) => (n || '').trim().toLowerCase() === reporterRaw,
        )
      // Manager-reported cycle: wait on that manager by name.
      if (reporterIsManager) return `Waiting for ${reporter} to finalize it`
      if (managers) return `Waiting for ${managers} to finalize it`
      if (actor) return `Waiting for ${actor} to finalize it`
      return 'Waiting for a manager to finalize it'
    }
    case 'done':
    case 'accepted':
    case 'accept':
      if (people.closedAt) {
        return `The task was closed in ${toShamsiDate(people.closedAt)}`
      }
      return 'The task was closed'
    case 'rejected':
    case 'reject':
      if (actor) return `Rejected by ${actor}`
      return 'Rejected'
    case 'edit':
      if (actor) return `Edited by ${actor}`
      return 'Edited'
    case 'team_message':
      if (actor) return `A comment from ${actor}`
      return 'A comment'
    default:
      return status
  }
}
