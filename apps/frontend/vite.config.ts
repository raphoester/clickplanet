import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'
import glsl from 'vite-plugin-glsl'

export default defineConfig({
    plugins: [react(), glsl()],
    base: "/",
    test: {
        include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
        /**
         * Node by default: most of the suite is domain logic with no DOM. The
         * few files that render components opt in with a `@vitest-environment`
         * comment, so the rest are not slowed down by a jsdom per file.
         */
        environment: "node",
    },
})
