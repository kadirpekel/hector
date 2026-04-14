/// <reference types="vite/client" />
declare const __APP_VERSION__: string

interface ImportMetaEnv {
  readonly VITE_HECTOR_CLOUD_URL?: string
  readonly VITE_CLOUD_SIGNUP_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
