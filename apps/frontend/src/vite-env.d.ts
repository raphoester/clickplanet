/// <reference types="vite/client" />

/**
 * vite-plugin-glsl inlines these as strings at build time. It ships an
 * ext.d.ts saying so, but nothing pulls that file into the program, so the
 * shader imports each carried a `@ts-expect-error` and typed as `any`.
 */
declare module '*.glsl' {
    const shader: string
    export default shader
}

interface ImportMetaEnv {
    readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
