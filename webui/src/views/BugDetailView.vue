<script setup lang="ts">
import PriorityLabel from '@/components/PriorityLabel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import TeamChatPanel from '@/components/TeamChatPanel.vue'
import Button from '@/components/ui/Button.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { api, type BugAttachment, type BugReport } from '@/lib/api'
import { getToken, getUser } from '@/lib/auth'
import { canAdmin, canApprove, canDevelop, canTesterReview } from '@/lib/permissions'
import { useRoleDirectory } from '@/lib/roleDirectory'
import { autoDir, applyBugBaseUrl, toShamsiDateTime } from '@/lib/format'
import { getBugBaseUrlOverride } from '@/lib/settings'
import { badgeActorUsername, badgeStatus, displayPersonName } from '@/lib/status'
import { Check, ChevronLeft, ChevronRight, Download, Pencil, Trash2, Wrench, X } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const user = getUser()
const bug = ref<BugReport | null>(null)
const attachments = ref<BugAttachment[]>([])
const historyRows = ref<
  Array<{
    id: number
    url: string
    status: string
    rejection_reason: string | null
    created_at: string
    actor_username: string
    reporter_username: string
  }>
>([])
const error = ref('')
const loading = ref(true)
const busy = ref(false)
const rejectReason = ref('')
const rejectDir = computed(() => autoDir(rejectReason.value))
const showReject = ref(false)
const editing = ref(false)
const editDescription = ref('')
const editPageParameters = ref('')
const editDescDir = computed(() => autoDir(editDescription.value))
const editParamsDir = computed(() => autoDir(editPageParameters.value))
const { developerNames, testerNames, managerNames, ensureRoleDirectory } = useRoleDirectory()

const bugId = computed(() => Number(route.params.id))
const neighborIds = ref<number[]>([])
const prevBugId = computed(() => {
  const idx = neighborIds.value.indexOf(bugId.value)
  return idx > 0 ? neighborIds.value[idx - 1] : null
})
const nextBugId = computed(() => {
  const idx = neighborIds.value.indexOf(bugId.value)
  return idx >= 0 && idx < neighborIds.value.length - 1 ? neighborIds.value[idx + 1] : null
})

async function loadNeighborIds() {
  try {
    const rows = await api.bugReports()
    neighborIds.value = rows.map((r) => r.id)
  } catch {
    neighborIds.value = []
  }
}

function goPrevBug() {
  if (prevBugId.value != null) router.push({ name: 'bug-detail', params: { id: prevBugId.value } })
}

function goNextBug() {
  if (nextBugId.value != null) router.push({ name: 'bug-detail', params: { id: nextBugId.value } })
}

const imagePreviewUrls = ref<Record<number, string>>({})
const lightboxAtt = ref<BugAttachment | null>(null)
let previewSeq = 0

const imageAttachments = computed(() =>
  attachments.value.filter((a) => (a.content_type || '').toLowerCase().startsWith('image/')),
)
const fileAttachments = computed(() =>
  attachments.value.filter((a) => !(a.content_type || '').toLowerCase().startsWith('image/')),
)

function revokePreviewUrls() {
  for (const url of Object.values(imagePreviewUrls.value)) {
    URL.revokeObjectURL(url)
  }
  imagePreviewUrls.value = {}
}

async function loadImagePreviews(list: BugAttachment[]) {
  const seq = ++previewSeq
  revokePreviewUrls()
  const token = getToken()
  const next: Record<number, string> = {}
  await Promise.all(
    list.map(async (att) => {
      try {
        const url = api.attachmentDownloadUrl(bugId.value, att.id)
        const res = await fetch(url, {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        })
        if (!res.ok) return
        const objectUrl = URL.createObjectURL(await res.blob())
        if (seq !== previewSeq) {
          URL.revokeObjectURL(objectUrl)
          return
        }
        next[att.id] = objectUrl
      } catch {
        /* skip broken preview */
      }
    }),
  )
  if (seq !== previewSeq) {
    for (const url of Object.values(next)) URL.revokeObjectURL(url)
    return
  }
  imagePreviewUrls.value = next
}

function openLightbox(att: BugAttachment) {
  lightboxAtt.value = att
}

function closeLightbox() {
  lightboxAtt.value = null
}

function onLightboxKey(e: KeyboardEvent) {
  if (e.key === 'Escape') closeLightbox()
}

function downloadLightbox() {
  if (lightboxAtt.value) void downloadAttachment(lightboxAtt.value)
}

async function load() {
  loading.value = true
  error.value = ''
  lightboxAtt.value = null
  try {
    await ensureRoleDirectory()
    const id = bugId.value
    ;[bug.value, attachments.value, historyRows.value] = await Promise.all([
      api.bugReport(id),
      api.attachments(id),
      api.history(id),
    ])
    const images = attachments.value.filter((a) =>
      (a.content_type || '').toLowerCase().startsWith('image/'),
    )
    void loadImagePreviews(images)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function markFixed() {
  busy.value = true
  try {
    await api.updateStatus(bugId.value, 'fixed')
    await load()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Update failed'
  } finally {
    busy.value = false
  }
}

async function confirmReject(status: 'rejected') {
  if (!rejectReason.value.trim()) {
    error.value = 'Rejection reason is required'
    return
  }
  busy.value = true
  try {
    await api.updateStatus(bugId.value, status, rejectReason.value.trim())
    showReject.value = false
    rejectReason.value = ''
    await load()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Reject failed'
  } finally {
    busy.value = false
  }
}

async function testerApprove() {
  busy.value = true
  try {
    await api.updateStatus(bugId.value, 'waiting_manager')
    await load()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Approve failed'
  } finally {
    busy.value = false
  }
}

async function managerAccept() {
  busy.value = true
  try {
    await api.updateStatus(bugId.value, 'done')
    await load()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Accept failed'
  } finally {
    busy.value = false
  }
}

async function deleteBug() {
  if (!bug.value) return
  if (!window.confirm(`Delete bug #${bug.value.id}? This cannot be undone.`)) return
  busy.value = true
  try {
    await api.deleteBug(bugId.value)
    router.push({ name: 'bugs' })
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
    busy.value = false
  }
}

async function downloadAttachment(att: BugAttachment) {
  const token = getToken()
  const url = api.attachmentDownloadUrl(bugId.value, att.id)
  const res = await fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
  if (!res.ok) {
    error.value = 'Download failed'
    return
  }
  const blob = await res.blob()
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = att.original_filename
  a.click()
  URL.revokeObjectURL(a.href)
}

function startEdit() {
  if (!bug.value) return
  editDescription.value = bug.value.description
  editPageParameters.value = bug.value.page_parameters
  editing.value = true
}

function cancelEdit() {
  editing.value = false
}

async function saveEdit() {
  if (!editDescription.value.trim() || !editPageParameters.value.trim()) {
    error.value = 'Description and Page parameters are required'
    return
  }
  busy.value = true
  try {
    await api.updateBugFields(bugId.value, {
      description: editDescription.value.trim(),
      page_parameters: editPageParameters.value.trim(),
    })
    editing.value = false
    await load()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Save failed'
  } finally {
    busy.value = false
  }
}

const canDevFix = computed(
  () => canDevelop(user?.role) && bug.value?.status === 'reported',
)
const canTesterAct = computed(
  () =>
    canTesterReview(user?.role) &&
    bug.value?.status === 'fixed' &&
    (user?.role === 'admin' || user?.id === bug.value?.reporter_user_id),
)
const canManagerReview = computed(
  () => canApprove(user?.role) && bug.value?.status === 'waiting_manager',
)
const canAdminDelete = computed(() => canAdmin(user?.role))
const canEditFields = computed(
  () => !!user && !!bug.value && user.id === bug.value.reporter_user_id,
)

const bugHref = computed(() => applyBugBaseUrl(bug.value?.url || '', getBugBaseUrlOverride()))
const bugLinkLabel = computed(() => bugHref.value || bug.value?.url || '')
const statusBadge = computed(() => (bug.value ? badgeStatus(bug.value) : ''))
const statusActor = computed(() => (bug.value ? badgeActorUsername(bug.value) : undefined))

watch(bugId, () => {
  editing.value = false
  void load()
})

onMounted(() => {
  window.addEventListener('keydown', onLightboxKey)
  void loadNeighborIds()
  void load()
})

onUnmounted(() => {
  window.removeEventListener('keydown', onLightboxKey)
  revokePreviewUrls()
})
</script>

<template>
  <div class="flex h-[calc(100vh)] w-full flex-col overflow-hidden px-4 py-4">
    <div class="mb-3 flex shrink-0 flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          class="h-8 inline-flex items-center gap-1.5"
          :disabled="prevBugId == null || loading"
          @click="goPrevBug"
        >
          <ChevronLeft class="h-3.5 w-3.5" :stroke-width="1.75" />
          Previous
        </Button>
        <Button
          variant="outline"
          class="h-8 inline-flex items-center gap-1.5"
          :disabled="nextBugId == null || loading"
          @click="goNextBug"
        >
          Next
          <ChevronRight class="h-3.5 w-3.5" :stroke-width="1.75" />
        </Button>
      </div>
    </div>
    <p v-if="error" class="mb-2 shrink-0 text-rose-400">{{ error }}</p>
    <p v-if="loading" class="text-muted-foreground">Loading…</p>
    <div
      v-else-if="bug"
      class="grid min-h-0 flex-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(220px,280px)_minmax(320px,380px)] lg:items-stretch lg:overflow-hidden"
    >
      <div class="min-w-0 space-y-6 lg:min-h-0 lg:overflow-y-auto">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-2 text-sm text-muted-foreground">
          <span class="text-foreground">Status:</span>
          <StatusBadge
            :status="statusBadge"
            :fixer-username="bug.fixer_username"
            :reporter-username="bug.reporter_username"
            :actor-username="statusActor"
            :developer-names="developerNames"
            :tester-names="testerNames"
            :manager-names="managerNames"
            :closed-at="bug.status === 'done' ? bug.updated_at : null"
          />
          <span aria-hidden="true">|</span>
          <span class="inline-flex items-center gap-1.5">
            Priority: <PriorityLabel :priority="bug.priority" />
          </span>
          <span aria-hidden="true">|</span>
          <span
            >Created: <span class="font-mono text-xs text-foreground">{{
              toShamsiDateTime(bug.created_at)
            }}</span></span
          >
          <span aria-hidden="true">|</span>
          <span
            >Updated: <span class="font-mono text-xs text-foreground">{{
              toShamsiDateTime(bug.updated_at)
            }}</span></span
          >
        </div>

        <div class="space-y-3">
          <section class="field-panel-stack">
            <h2 class="field-panel-label">URL</h2>
            <a
              :href="bugHref"
              target="_blank"
              rel="noopener"
              class="text-sm text-sky-700 dark:text-sky-300 break-all hover:underline"
            >
              {{ bugLinkLabel }}
            </a>
          </section>

          <section class="field-panel-stack">
            <div class="flex items-center justify-between gap-2">
              <h2 class="field-panel-label">Description</h2>
              <Button
                v-if="canEditFields && !editing"
                variant="outline"
                class="h-7 inline-flex items-center gap-1.5"
                @click="startEdit"
              >
                <Pencil class="h-3 w-3" :stroke-width="1.75" />
                Edit
              </Button>
            </div>
            <Textarea
              v-if="editing"
              v-model="editDescription"
              :dir="editDescDir"
              :rows="8"
              class="min-h-[120px]"
            />
            <p
              v-else
              class="text-sm whitespace-pre-wrap text-muted-foreground"
              :dir="autoDir(bug.description)"
            >
              {{ bug.description }}
            </p>
          </section>

          <section class="field-panel-stack">
            <h2 class="field-panel-label">Page parameters</h2>
            <Textarea
              v-if="editing"
              v-model="editPageParameters"
              :dir="editParamsDir"
              :rows="4"
            />
            <pre
              v-else
              class="text-xs overflow-x-auto whitespace-pre-wrap text-muted-foreground"
              :dir="autoDir(bug.page_parameters)"
            >{{ bug.page_parameters }}</pre>
          </section>

          <div v-if="editing" class="flex flex-wrap gap-2">
            <Button class="h-8" :disabled="busy" @click="saveEdit">Save</Button>
            <Button variant="outline" class="h-8" :disabled="busy" @click="cancelEdit">Cancel</Button>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <section class="field-panel-row">
              <span class="field-panel-label">Reporter</span>
              <p class="field-panel-value">{{ displayPersonName(bug.reporter_username) }}</p>
            </section>

            <section class="field-panel-row">
              <span class="field-panel-label">Fixer</span>
              <p class="field-panel-value">{{ displayPersonName(bug.fixer_username) || '—' }}</p>
            </section>
          </div>

          <section v-if="bug.rejection_reason" class="field-panel-stack">
            <span class="field-panel-label">Last rejection reason</span>
            <p class="text-sm text-muted-foreground" :dir="autoDir(bug.rejection_reason)">
              {{ bug.rejection_reason }}
            </p>
          </section>
        </div>

        <section class="space-y-3">
          <h2 class="text-sm font-medium text-muted-foreground">Attachments</h2>
          <div v-if="imageAttachments.length" class="grid grid-cols-2 gap-2 sm:grid-cols-3">
            <button
              v-for="att in imageAttachments"
              :key="att.id"
              type="button"
              class="group relative aspect-square overflow-hidden rounded-md border border-border bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              :title="att.original_filename"
              @click="openLightbox(att)"
            >
              <img
                v-if="imagePreviewUrls[att.id]"
                :src="imagePreviewUrls[att.id]"
                :alt="att.original_filename"
                class="h-full w-full object-cover transition-transform group-hover:scale-105"
              />
              <span
                v-else
                class="flex h-full w-full items-center justify-center px-2 text-center text-xs text-muted-foreground"
              >
                Loading…
              </span>
            </button>
          </div>
          <ul v-if="fileAttachments.length" class="space-y-2">
            <li
              v-for="att in fileAttachments"
              :key="att.id"
              class="flex flex-wrap items-center gap-3 text-sm"
            >
              <span class="truncate max-w-md" :title="att.original_filename">{{
                att.original_filename
              }}</span>
              <span class="text-xs text-muted-foreground">{{ att.size_bytes }} bytes</span>
              <Button
                variant="outline"
                class="h-8 inline-flex items-center gap-1.5"
                @click="downloadAttachment(att)"
              >
                <Download class="h-3.5 w-3.5" :stroke-width="1.75" />
                Download
              </Button>
            </li>
          </ul>
          <p
            v-if="!imageAttachments.length && !fileAttachments.length"
            class="text-sm text-muted-foreground"
          >
            No attachments
          </p>
        </section>

        <section
          v-if="canDevFix || canTesterAct || canManagerReview || canAdminDelete"
          class="space-y-3 border-t border-border pt-4"
        >
          <div class="flex flex-wrap gap-2">
            <Button
              v-if="canDevFix"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="markFixed"
            >
              <Wrench class="h-3.5 w-3.5" :stroke-width="1.75" />
              Mark fixed
            </Button>
            <Button
              v-if="canTesterAct"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="testerApprove"
            >
              <Check class="h-3.5 w-3.5" :stroke-width="1.75" />
              Approve
            </Button>
            <Button
              v-if="canTesterAct"
              variant="destructive"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="showReject = true"
            >
              <X class="h-3.5 w-3.5" :stroke-width="1.75" />
              Reject (back to fixer)
            </Button>
            <Button
              v-if="canManagerReview"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="managerAccept"
            >
              <Check class="h-3.5 w-3.5" :stroke-width="1.75" />
              Finalize
            </Button>
            <Button
              v-if="canManagerReview"
              variant="destructive"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="showReject = true"
            >
              <X class="h-3.5 w-3.5" :stroke-width="1.75" />
              Reject (back to fixer)
            </Button>
            <Button
              v-if="canAdminDelete"
              variant="destructive"
              class="inline-flex items-center gap-1.5"
              :disabled="busy"
              @click="deleteBug"
            >
              <Trash2 class="h-3.5 w-3.5" :stroke-width="1.75" />
              Delete bug
            </Button>
          </div>
          <div v-if="showReject" class="space-y-2 max-w-md">
            <Textarea v-model="rejectReason" :dir="rejectDir" placeholder="Reason of rejection *" />
            <Button class="h-8" :disabled="busy" @click="confirmReject('rejected')">Confirm reject</Button>
          </div>
        </section>
      </div>

      <section class="flex min-h-0 flex-col border-t border-border pt-4 lg:border-t-0 lg:border-s lg:ps-4 lg:pt-0">
        <h2 class="mb-3 shrink-0 text-sm font-medium">Timeline</h2>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <ol v-if="historyRows.length" class="relative space-y-0 border-s border-border ms-2">
            <li v-for="row in historyRows" :key="row.id" class="ms-4 pb-6 last:pb-0">
              <span
                class="absolute -start-1.5 mt-1.5 h-3 w-3 rounded-full border border-border bg-card"
              />
              <div class="space-y-1">
                <StatusBadge
                  :status="row.status"
                  :actor-username="row.actor_username"
                  :reporter-username="row.reporter_username"
                  :developer-names="developerNames"
                  :tester-names="testerNames"
                  :manager-names="managerNames"
                  :closed-at="row.status === 'done' ? row.created_at : null"
                />
                <p class="text-xs text-muted-foreground">
                  {{ displayPersonName(row.actor_username) }}
                  ·
                  <span class="font-mono">{{ toShamsiDateTime(row.created_at) }}</span>
                </p>
                <p
                  v-if="row.rejection_reason"
                  class="text-sm text-muted-foreground"
                  :dir="autoDir(row.rejection_reason)"
                >
                  {{ row.rejection_reason }}
                </p>
              </div>
            </li>
          </ol>
          <p v-else class="text-sm text-muted-foreground">No history yet</p>
        </div>
      </section>

      <TeamChatPanel :bug-id="bug.id" class="lg:min-h-0" @message-sent="load" />
    </div>

    <div
      v-if="lightboxAtt"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
      role="dialog"
      aria-modal="true"
      :aria-label="lightboxAtt.original_filename"
      @click.self="closeLightbox"
    >
      <div class="flex max-h-full max-w-5xl flex-col gap-3">
        <img
          v-if="imagePreviewUrls[lightboxAtt.id]"
          :src="imagePreviewUrls[lightboxAtt.id]"
          :alt="lightboxAtt.original_filename"
          class="max-h-[80vh] max-w-full object-contain"
        />
        <div class="flex flex-wrap items-center justify-between gap-2 text-sm text-white">
          <span class="truncate">{{ lightboxAtt.original_filename }}</span>
          <div class="flex gap-2">
            <Button
              variant="outline"
              class="h-8 inline-flex items-center gap-1.5"
              @click="downloadLightbox"
            >
              <Download class="h-3.5 w-3.5" :stroke-width="1.75" />
              Download
            </Button>
            <Button variant="outline" class="h-8" @click="closeLightbox">Close</Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
