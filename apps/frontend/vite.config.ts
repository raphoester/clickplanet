import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'
import glsl from 'vite-plugin-glsl'

export default defineConfig({
    plugins: [react(), glsl()],
    base: "/",
    test: {
        include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
    },
})
