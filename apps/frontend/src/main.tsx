import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './index.css'

import {newClickServiceClient, newSessionServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient} from "./backends/chatBackend.ts"
import {FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {SnapshotBackend} from "./backends/snapshotBackend.ts"
import App from "./app/App.tsx"

const config = {
    baseUrl: import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol",
    timeoutMs: 2000,
}

// Without a sitekey the client sends no session, which is what a local backend
// with sessions off expects. A server that enforces them refuses every click
// from such a build, deliberately: the two are configured together.
const sitekey = import.meta.env.VITE_TURNSTILE_SITEKEY
const session: SessionProvider = sitekey
    ? new SessionClient(newSessionServiceClient(config), turnstileAttester(sitekey, "session"))
    : new NoSession()

// SCRATCH R&D: VITE_DEV_SNAPSHOT serves a frozen production map from
// public/dev-map.bin, which is the only way to look at realistic ownership from
// localhost — the real API's CORS header names the deployed origin only.
const snapshot = import.meta.env.VITE_DEV_SNAPSHOT ? new SnapshotBackend() : undefined

const backend = snapshot ?? new PlanetBackend(config, newClickServiceClient(config), 100, session)
const chatBackend = snapshot
    ? new FakeChatBackend({unavailable: true})
    : new ChatServiceBackend(config, newChatServiceClient(config))

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <App
            ownershipsGetter={backend}
            tileClicker={backend}
            updatesListener={backend}
            chatBackend={chatBackend}
        />
    </StrictMode>,
)
