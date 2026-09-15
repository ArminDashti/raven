import type { Role } from './auth'

/** Report a bug */
export function canReport(role: Role | string | undefined | null): boolean {
  return role === 'tester' || role === 'developer' || role === 'manager' || role === 'admin'
}

/** Mark fixed / work the bug */
export function canDevelop(role: Role | string | undefined | null): boolean {
  return role === 'developer' || role === 'manager' || role === 'admin'
}

/** Tester-stage approve / reject after Dev marks fixed */
export function canTesterReview(role: Role | string | undefined | null): boolean {
  return role === 'tester' || role === 'admin'
}

/** Approve or reject at manager stage */
export function canApprove(role: Role | string | undefined | null): boolean {
  return role === 'manager' || role === 'admin'
}

/** Delete bugs and full admin */
export function canAdmin(role: Role | string | undefined | null): boolean {
  return role === 'admin'
}

export function isReporterRole(role: Role | string | undefined | null): boolean {
  return canReport(role)
}
