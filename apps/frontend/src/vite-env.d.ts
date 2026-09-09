/// <reference types="vite/client" />

declare module '*.glsl' {
    const shader: string
    export default shader
}

interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string
    // The Turnstile widget's sitekey. Public — it is read off the page — and
    // useless without the secret, which only the backend holds. Unset means
    // this build sends no session.
    readonly VITE_TURNSTILE_SITEKEY?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
