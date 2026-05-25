/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_GAME_SERVER_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

declare global {
  namespace NodeJS {
    interface ProcessEnv {
      readonly SENTRY_AUTH_TOKEN?: string;
      readonly VITE_GAME_SERVER_URL?: string;
    }
  }
}

export {};
