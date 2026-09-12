import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './index.css'

import {newClickServiceClient, newSessionServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient} from "./backends/chatBackend.ts"
import {FakeBackend} from "./backends/fakeBackend.ts"
import {FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import type {Globe} from "./app/viewer/globe.ts"
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

// `VITE_FAKE_BACKEND=1 npm run dev` plays against the in-browser fakes, bombs
// included. Dev only: `DEV` is folded to false in a build, which is what drops
// both fakes from the bundle — an unset `VITE_*` variable is not folded.
const root = createRoot(document.getElementById('root')!)

if (import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND) {
    const fake = new FakeBackend(100, {
        tilePositions: () => loadPointGeometryData().then((data) => data.positions),
    })
    // Console commands:
    // - `giveBomb()`: as if you had just caught a box and it held a bomb.
    // - `fakeBackend.botBomb(tile, "fr")`: someone else's bomb lands on `tile`.
    Object.assign(window, {
        fakeBackend: fake,
        giveBomb: () => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBomb())
            return "💣 armed — press and hold on the planet"
        },
    })

    root.render(
        <StrictMode>
            <App
                ownershipsGetter={fake}
                tileClicker={fake}
                updatesListener={fake}
                bonusListener={fake}
                bomber={fake}
                clickBudgetSource={fake}
                chatBackend={new FakeChatBackend()}
            />
        </StrictMode>,
    )
} else {
    const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
    const chatBackend = new ChatServiceBackend(newChatServiceClient(config))

    root.render(
        <StrictMode>
            <App
                ownershipsGetter={backend}
                tileClicker={backend}
                updatesListener={backend}
                bonusListener={backend}
                bomber={backend}
                clickBudgetSource={backend}
                chatBackend={chatBackend}
            />
        </StrictMode>,
    )
}
