import {defineConfig, Plugin} from 'vitest/config'
import react from '@vitejs/plugin-react'
import glsl from 'vite-plugin-glsl'

/**
 * Two pages: index.html is the home page, play.html is the game. The game is
 * served at /play and at /auth/callback, the redirect URI registered with Google
 * and Discord.
 *
 * The Workers asset handler serves a real file at a path before its
 * single-page-application fallback, and that fallback is index.html — the home
 * page. So the build writes the game a second time as auth/callback.html, which
 * the handler serves at /auth/callback. The dev server gets the same routes from
 * a middleware.
 */
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
            if (game?.type !== "asset") this.error(`${GAME} is missing from the bundle`)
            this.emitFile({type: "asset", fileName: CALLBACK, source: game.source})
        },
    }
}

export default defineConfig({
    plugins: [react(), glsl(), gameRoutes()],
    base: "/",
    build: {
        rollupOptions: {
            input: {home: "index.html", play: "play.html"},
        },
    },
    test: {
        // `scripts/` is in here for the map generators alone. What they decide — where every tile
        // is and who owns the ground under it — outlives any one run and is not visible in a diff,
        // so their rules are pinned like the app's own.
        include: ["src/**/*.test.ts", "src/**/*.test.tsx", "scripts/**/*.test.mjs"],
        environment: "node",
    },
})
