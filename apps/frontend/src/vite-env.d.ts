/// <reference types="vite/client" />

declare module '*.html?raw' {
    const html: string
    export default html
}

declare module '*.glsl' {
    const shader: string
    export default shader
}

interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string
    readonly VITE_TURNSTILE_SITEKEY?: string
    readonly VITE_FAKE_BACKEND?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
