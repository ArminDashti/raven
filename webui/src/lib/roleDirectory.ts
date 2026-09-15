import { api } from './api'
import { fullNameFromParts, setPersonNameLookup } from './status'
import { ref } from 'vue'

const loaded = ref(false)
const developerNames = ref<string[]>([])
const testerNames = ref<string[]>([])
const managerNames = ref<string[]>([])
let inflight: Promise<void> | null = null

export async function ensureRoleDirectory(): Promise<void> {
  if (loaded.value) return
  if (inflight) return inflight
  inflight = (async () => {
    try {
      const users = await api.users()
      const lookup: Record<string, string> = {}
      for (const u of users) {
        lookup[u.username.toLowerCase()] = fullNameFromParts(u.first_name, u.last_name, u.username)
      }
      setPersonNameLookup(lookup)
      developerNames.value = users.filter((u) => u.role === 'developer').map((u) => u.username)
      testerNames.value = users.filter((u) => u.role === 'tester').map((u) => u.username)
      managerNames.value = users.filter((u) => u.role === 'manager').map((u) => u.username)
      loaded.value = true
    } finally {
      inflight = null
    }
  })()
  return inflight
}

export function useRoleDirectory() {
  return { developerNames, testerNames, managerNames, ensureRoleDirectory }
}
