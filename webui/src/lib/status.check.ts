/**
 * Runnable check for badgeStatus / statusLabel team_message (ponytail).
 * Run: npx --yes tsx src/lib/status.check.ts
 */
import { badgeActorUsername, badgeStatus, statusLabel } from './status'

const cases: Array<{
  name: string
  ok: boolean
}> = [
  {
    name: 'badge prefers team_message',
    ok:
      badgeStatus({
        status: 'reported',
        last_history_status: 'team_message',
        last_history_actor: 'developer',
      }) === 'team_message',
  },
  {
    name: 'badge actor from last history',
    ok:
      badgeActorUsername({
        status: 'reported',
        last_history_status: 'team_message',
        last_history_actor: 'developer',
      }) === 'developer',
  },
  {
    name: 'label uses first/last via actor display',
    ok:
      statusLabel('team_message', { actorUsername: 'Alex Dev' }) ===
      'A comment from Alex Dev',
  },
  {
    name: 'workflow status when last history is not comment',
    ok:
      badgeStatus({
        status: 'fixed',
        last_history_status: 'fixed',
        last_history_actor: 'dev',
      }) === 'fixed',
  },
]

let failed = 0
for (const c of cases) {
  if (!c.ok) {
    console.error(`FAIL ${c.name}`)
    failed++
  }
}
if (failed) {
  console.error(`${failed} check(s) failed`)
  process.exit(1)
}
console.log('status team_message: ok')
