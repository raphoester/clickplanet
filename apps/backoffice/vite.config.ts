import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'

// The port is the backend's auth.adminGate.origin: the gate refuses any other origin.
const PORT = 5174

// `vite --mode demo` answers every call in the page, from src/demo, in place of the backend.
const DEMO_API = '/src/demo/api.ts'

export default defineConfig(({mode}) => ({
    plugins: [react()],
    resolve: mode === 'demo' ? {alias: [{find: /^(\.\.?\/)+api\.ts$/, replacement: DEMO_API}]} : undefined,
    server: {port: PORT, strictPort: true},
    preview: {port: PORT, strictPort: true},
    test: {
        include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
        environment: "node",
    },
}))
