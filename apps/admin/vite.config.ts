import {defineConfig} from 'vitest/config'
import react from '@vitejs/plugin-react'

// The port is the backend's auth.adminGate.origin: the gate refuses any other origin.
const PORT = 5174

export default defineConfig({
    plugins: [react()],
    server: {port: PORT, strictPort: true},
    preview: {port: PORT, strictPort: true},
    test: {
        include: ["src/**/*.test.ts"],
        environment: "node",
    },
})
