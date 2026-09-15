<script setup lang="ts">
import { getToken } from '@/lib/auth'
import { api } from '@/lib/api'
import { hashName, resolveAvatarKind } from '@/lib/avatar'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

const HAIR = ['#2C2C2C', '#5D4037', '#EF6C00', '#F9A825', '#6A1B9A', '#00838F', '#37474F', '#CC7832']
const SHIRT = ['#365880', '#6A8759', '#CC7832', '#9876AA', '#6897BB', '#B5BD68', '#FFC66D', '#4A88C7']
const SKIN = ['#F8D5B8', '#E8B992', '#D4A574', '#C09070']
const BG = ['#3C3F41', '#2B2B2B', '#45494A', '#4C5052']

const props = withDefaults(
  defineProps<{
    gender?: string | null
    avatarStyle?: string | null
    name?: string | null
    userId?: number | null
    hasAvatar?: boolean
    size?: 'xs' | 'sm' | 'md' | 'lg'
  }>(),
  { gender: '', avatarStyle: '', name: '', userId: null, hasAvatar: false, size: 'md' },
)

const photoUrl = ref('')
let photoSeq = 0

const kind = computed(() => resolveAvatarKind(props.avatarStyle, props.gender, props.name))

const palette = computed(() => {
  const h = hashName(`${props.name || ''}::${kind.value}`)
  return {
    hair: HAIR[h % HAIR.length],
    shirt: SHIRT[(h >>> 3) % SHIRT.length],
    skin: SKIN[(h >>> 6) % SKIN.length],
    bg: BG[(h >>> 9) % BG.length],
    glasses: (h >>> 11) % 3 === 0 || kind.value === 'nerd',
    wink: (h >>> 13) % 5 === 0,
    blush: kind.value === 'woman' || (h >>> 15) % 4 === 0,
  }
})

const box = computed(() => {
  if (props.size === 'xs') return 'h-4 w-4'
  if (props.size === 'sm') return 'h-8 w-8'
  if (props.size === 'lg') return 'h-16 w-16'
  return 'h-10 w-10'
})

const title = computed(() => {
  if (photoUrl.value) {
    const who = (props.name || '').trim()
    return who ? `${who} (photo)` : 'Avatar photo'
  }
  if (kind.value === 'empty') return 'Avatar'
  const who = (props.name || '').trim()
  const role = kind.value.charAt(0).toUpperCase() + kind.value.slice(1)
  return who ? `${who} (${role})` : role
})

async function loadPhoto() {
  const seq = ++photoSeq
  if (photoUrl.value) {
    URL.revokeObjectURL(photoUrl.value)
    photoUrl.value = ''
  }
  if (!props.hasAvatar || !props.userId) return
  try {
    const token = getToken()
    const res = await fetch(api.userAvatarUrl(props.userId), {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) return
    const url = URL.createObjectURL(await res.blob())
    if (seq !== photoSeq) {
      URL.revokeObjectURL(url)
      return
    }
    photoUrl.value = url
  } catch {
    /* keep cartoon */
  }
}

watch(
  () => [props.userId, props.hasAvatar] as const,
  () => {
    void loadPhoto()
  },
)

onMounted(() => {
  void loadPhoto()
})

onUnmounted(() => {
  photoSeq++
  if (photoUrl.value) URL.revokeObjectURL(photoUrl.value)
})
</script>

<template>
  <span
    class="inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full border border-border"
    :class="box"
    :title="title"
    role="img"
    :aria-label="title"
  >
    <img v-if="photoUrl" :src="photoUrl" alt="" class="h-full w-full object-cover" />
    <svg
      v-else-if="kind === 'empty'"
      viewBox="0 0 24 24"
      class="h-full w-full p-[20%] text-muted-foreground"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linecap="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="9" />
    </svg>
    <svg v-else-if="kind === 'robot'" viewBox="0 0 64 64" class="h-full w-full" aria-hidden="true">
      <circle cx="32" cy="32" r="32" :fill="palette.bg" />
      <rect x="16" y="18" width="32" height="28" rx="6" fill="#90A4AE" />
      <circle cx="24" cy="30" r="4" fill="#4FC3F7" />
      <circle cx="40" cy="30" r="4" fill="#4FC3F7" />
      <rect x="24" y="40" width="16" height="3" rx="1.5" fill="#37474F" />
      <line x1="32" y1="12" x2="32" y2="18" stroke="#78909C" stroke-width="3" />
      <circle cx="32" cy="10" r="3" fill="#EF5350" />
    </svg>
    <svg v-else-if="kind === 'fox'" viewBox="0 0 64 64" class="h-full w-full" aria-hidden="true">
      <circle cx="32" cy="32" r="32" :fill="palette.bg" />
      <ellipse cx="32" cy="58" rx="20" ry="12" fill="#EF6C00" />
      <circle cx="32" cy="30" r="16" fill="#FFA726" />
      <path d="M16 22 L22 8 L28 24 Z" fill="#EF6C00" />
      <path d="M36 24 L42 8 L48 22 Z" fill="#EF6C00" />
      <circle cx="26" cy="30" r="2.2" fill="#2B2B2B" />
      <circle cx="38" cy="30" r="2.2" fill="#2B2B2B" />
      <path d="M28 38 L32 42 L36 38" fill="none" stroke="#BF360C" stroke-width="2" stroke-linecap="round" />
    </svg>
    <svg v-else-if="kind === 'abstract'" viewBox="0 0 64 64" class="h-full w-full" aria-hidden="true">
      <circle cx="32" cy="32" r="32" :fill="palette.bg" />
      <polygon points="32,10 50,42 14,42" :fill="palette.shirt" />
      <circle cx="32" cy="34" r="10" :fill="palette.hair" />
      <rect x="22" y="44" width="20" height="6" rx="2" :fill="palette.skin" />
    </svg>
    <svg v-else viewBox="0 0 64 64" class="h-full w-full" aria-hidden="true">
      <circle cx="32" cy="32" r="32" :fill="palette.bg" />
      <ellipse cx="32" cy="58" rx="22" ry="14" :fill="palette.shirt" />
      <circle cx="32" cy="30" r="16" :fill="palette.skin" />
      <path
        v-if="kind === 'man' || kind === 'nerd'"
        :fill="palette.hair"
        d="M16.5 30c.6-11 7.2-18 15.5-18s14.9 7 15.5 18c-2.2-6.5-8-10.5-15.5-10.5S18.7 23.5 16.5 30Z"
      />
      <path
        v-else
        :fill="palette.hair"
        d="M15 28c1-12 8-20 17-20s16 8 17 20c-1 2-2 4-2 4 0 8-1.5 16-3.5 20h-4c1.2-6 1.8-12 1.8-18 0-7-4.8-12-9.3-12s-9.3 5-9.3 12c0 6 .6 12 1.8 18h-4c-2-4-3.5-12-3.5-20 0 0-1-2-2-4Z"
      />
      <ellipse v-if="palette.blush" cx="22.5" cy="34" rx="3.2" ry="2" fill="#F48FB1" opacity="0.55" />
      <ellipse v-if="palette.blush" cx="41.5" cy="34" rx="3.2" ry="2" fill="#F48FB1" opacity="0.55" />
      <circle v-if="!palette.wink" cx="26" cy="30" r="2.1" fill="#2B2B2B" />
      <path
        v-else
        d="M23.5 30.5h5"
        stroke="#2B2B2B"
        stroke-width="2"
        stroke-linecap="round"
        fill="none"
      />
      <circle cx="38" cy="30" r="2.1" fill="#2B2B2B" />
      <path
        d="M26 39c2.4 3 9.6 3 12 0"
        fill="none"
        stroke="#C0392B"
        stroke-width="2"
        stroke-linecap="round"
      />
      <g v-if="palette.glasses" fill="none" stroke="#2B2B2B" stroke-width="1.6">
        <circle cx="26" cy="30" r="4.4" />
        <circle cx="38" cy="30" r="4.4" />
        <path d="M30.4 30h3.2" />
      </g>
    </svg>
  </span>
</template>
