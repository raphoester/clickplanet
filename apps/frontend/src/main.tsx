import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './index.css'

import {newClickServiceClient, newSessionServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient} from "./backends/chatBackend.ts"
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

const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
const chatBackend = new ChatServiceBackend(newChatServiceClient(config))

createRoot(document.getElementById('root')!).render(
    <StrictMode>
        <App
            ownershipsGetter={backend}
            tileClicker={backend}
            updatesListener={backend}
            clickBudgetSource={backend}
            chatBackend={chatBackend}
        />
    </StrictMode>,
)
