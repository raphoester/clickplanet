import {defineConfig, Plugin} from 'vitest/config'
import react from '@vitejs/plugin-react'
import glsl from 'vite-plugin-glsl'
import {DISCORD_INVITE} from './src/links.ts'
import {WEB_ANALYTICS_TOKEN, withoutAnalytics} from './src/webAnalytics.ts'

function gameRoutes(): Plugin {
    const GAME = "play.html"
    const CALLBACK = "auth/callback.html"
    const ROUTES = ["/play", "/auth/callback"]

    return {
        name: "clickplanet-game-routes",
        enforce: "post",
        configureServer(server) {
            server.middlewares.use((req, _res, next) => {
                const url = new URL(req.url ?? "/", "http://localhost")
                if (ROUTES.includes(url.pathname.replace(/\/+$/, ""))) req.url = `/${GAME}${url.search}`
                next()
            })
        },
        generateBundle(_options, bundle) {
            const game = bundle[GAME]
            if (game?.type !== "asset" || typeof game.source !== "string") this.error(`${GAME} is missing from the bundle`)
            const callback = withoutAnalytics(game.source)
            if (callback.includes("cloudflareinsights")) this.error(`${CALLBACK} still loads the analytics beacon`)
            this.emitFile({type: "asset", fileName: CALLBACK, source: callback})
        },
    }
}

export default defineConfig({
    plugins: [react(), glsl(), gameRoutes()],
    base: "/",
    define: {
        "import.meta.env.DISCORD_INVITE": JSON.stringify(DISCORD_INVITE),
        "import.meta.env.WEB_ANALYTICS_TOKEN": JSON.stringify(WEB_ANALYTICS_TOKEN),
    },
    build: {
        rollupOptions: {
            input: {home: "index.html", play: "play.html", privacy: "privacy.html", terms: "terms.html"},
        },
    },
    test: {
        include: ["src/**/*.test.ts", "src/**/*.test.tsx", "scripts/**/*.test.mjs"],
        environment: "node",
    },
})
