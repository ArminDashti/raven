/**
 * Runnable check for name-based avatar gender.
 * Run: npx --yes tsx src/lib/avatar.check.ts
 */
import { avatarGenderForName } from './avatar'

const cases: Array<{ name: string; input: string; want: 'man' | 'woman' | '' }> = [
  { name: 'empty', input: '', want: '' },
  { name: 'alex display', input: 'Alex Dev', want: 'man' },
  { name: 'sam display', input: 'Sam Manager', want: 'woman' },
  { name: 'jordan display', input: 'Jordan Tester', want: 'woman' },
  { name: 'sam username', input: 'sam.manager', want: 'woman' },
  { name: 'manager login token', input: 'sam', want: 'woman' },
  { name: 'casey display', input: 'Casey Tester', want: 'man' },
  { name: 'maria display', input: 'Maria Demo', want: 'woman' },
]

let failed = 0
for (const c of cases) {
  const got = avatarGenderForName(c.input)
  if (got !== c.want) {
    console.error(`FAIL ${c.name}: got ${got || '(empty)'} want ${c.want || '(empty)'}`)
    failed++
  }
}
if (failed) {
  console.error(`${failed} check(s) failed`)
  process.exit(1)
}
console.log('avatar gender: ok')
