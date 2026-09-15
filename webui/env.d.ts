/// <reference types="vite/client" />
/// <reference types="vite-plugin-pwa/client" />

declare const __APP_VERSION__: string

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

declare module 'jalaali-js' {
  export function toJalaali(
    gy: number,
    gm: number,
    gd: number,
  ): { jy: number; jm: number; jd: number }
}
