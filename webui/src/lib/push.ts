import { api } from './api'

const DISMISS_KEY = 'raven-push-dismiss'

export type PushCapability =
  | 'unsupported'
  | 'insecure'
  | 'default'
  | 'denied'
  | 'granted'

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = window.atob(base64)
  const out = new Uint8Array(raw.length)
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i)
  return out
}

export function isPushApiAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  )
}

export function getNotificationPermission(): NotificationPermission | 'unsupported' {
  if (!('Notification' in window)) return 'unsupported'
  return Notification.permission
}

export function getPushCapability(): PushCapability {
  if (!isPushApiAvailable()) return 'unsupported'
  if (!window.isSecureContext) return 'insecure'
  const p = Notification.permission
  if (p === 'denied') return 'denied'
  if (p === 'granted') return 'granted'
  return 'default'
}

export function isPushBannerDismissed(): boolean {
  try {
    return sessionStorage.getItem(DISMISS_KEY) === '1'
  } catch {
    return false
  }
}

export function dismissPushBanner(): void {
  try {
    sessionStorage.setItem(DISMISS_KEY, '1')
  } catch {
    /* ignore */
  }
}

export function clearPushBannerDismiss(): void {
  try {
    sessionStorage.removeItem(DISMISS_KEY)
  } catch {
    /* ignore */
  }
}

async function getPushRegistration(): Promise<ServiceWorkerRegistration> {
  const scope = import.meta.env.BASE_URL
  const existing = await navigator.serviceWorker.getRegistration(scope)
  if (existing) return existing
  return navigator.serviceWorker.ready
}

/** Subscribe only when permission is already granted (no permission prompt). */
export async function subscribeAfterPermissionGranted(): Promise<boolean> {
  if (!isPushApiAvailable() || !window.isSecureContext) return false
  if (Notification.permission !== 'granted') return false

  try {
    const { publicKey } = await api.vapidPublicKey()
    if (!publicKey) return false

    // PWA SW is registered from main.ts (virtual:pwa-register); reuse it for push.
    const reg = await getPushRegistration()

    let sub = await reg.pushManager.getSubscription()
    if (!sub) {
      sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey),
      })
    }
    const json = sub.toJSON()
    if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false
    await api.pushSubscribe({
      endpoint: json.endpoint,
      keys: { p256dh: json.keys.p256dh, auth: json.keys.auth },
    })
    return true
  } catch {
    return false
  }
}

/**
 * Call from a user gesture (button click). Requests permission if needed, then subscribes.
 */
export async function ensurePushSubscription(): Promise<'granted' | 'denied' | 'default' | 'unavailable'> {
  if (!isPushApiAvailable()) return 'unavailable'
  if (!window.isSecureContext) return 'unavailable'

  let permission = Notification.permission
  if (permission === 'default') {
    permission = await Notification.requestPermission()
  }
  if (permission === 'denied') return 'denied'
  if (permission !== 'granted') return 'default'

  const ok = await subscribeAfterPermissionGranted()
  return ok ? 'granted' : 'unavailable'
}
