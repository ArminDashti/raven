/**
 * Runnable check for applyBugBaseUrl (ponytail).
 * Run: npx --yes tsx src/lib/format.baseurl.check.ts
 */
import { applyBugBaseUrl } from './format'

const sample =
  'http://erp.dpdc.co:8880/Pages/Warehouse/02-EtelaatePaieh/KalaAdamForoshGorohiVahedKharid.aspx'

const cases: Array<[string, string, string]> = [
  ['', sample, sample],
  [
    'http://example.com',
    sample,
    'http://example.com/Pages/Warehouse/02-EtelaatePaieh/KalaAdamForoshGorohiVahedKharid.aspx',
  ],
  [
    'http://example.com/',
    sample,
    'http://example.com/Pages/Warehouse/02-EtelaatePaieh/KalaAdamForoshGorohiVahedKharid.aspx',
  ],
  ['not-a-url', sample, sample],
]

let failed = 0
for (const [base, url, want] of cases) {
  const got = applyBugBaseUrl(url, base)
  if (got !== want) {
    console.error(`FAIL base=${JSON.stringify(base)}\n  got  ${got}\n  want ${want}`)
    failed++
  }
}
if (failed) {
  console.error(`${failed} check(s) failed`)
  process.exit(1)
}
console.log('applyBugBaseUrl: ok')
