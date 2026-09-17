import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './index.css'

import {newClickServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {newAuthServiceClient, SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient} from "./backends/chatBackend.ts"
import {FakeBackend} from "./backends/fakeBackend.ts"
import {FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import type {Globe} from "./app/viewer/globe.ts"
import App from "./app/App.tsx"
import {ConnectAccountBackend} from "./backends/accountBackend.ts"
import {AccountStore} from "./app/account/accountStore.ts"
import {rememberSignIn} from "./app/account/rememberedSignIn.ts"
import SignInGate from "./app/account/SignInGate.tsx"
import {callbackOf, CALLBACK_PATH} from "./domain/signInCallback.ts"

// The provider's code and state leave the address bar before anything else
// runs: nothing may bookmark, log, share or send them on as a referrer. The
// page they came in on already set `no-referrer` (see index.html).
const callback = callbackOf(new URL(window.location.href))
if (callback) window.history.replaceState(null, "", CALLBACK_PATH)

const config = {
    baseUrl: import.meta.env.VITE_API_BASE_URL ?? "https://api.clickplanet.lol",
    timeoutMs: 2000,
}

// Without a sitekey the client sends no session, which is what a local backend
// with sessions off expects. A server that enforces them refuses every click
// from such a build, deliberately: the two are configured together.
const sitekey = import.meta.env.VITE_TURNSTILE_SITEKEY
const authClient = newAuthServiceClient(config)
const session: SessionProvider = sitekey
    ? new SessionClient(authClient, turnstileAttester(sitekey, "session"))
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
    // - `giveBonus("tripleClicks")`: the same for any other bonus.
    // - `fakeBackend.botBomb(tile, "fr")`: someone else's bomb lands on `tile`.
    // - `fakeBackend.botSpread(tile, "fr")`, `fakeBackend.botBoost(tile, "fr")`:
    //   someone else's spread or boosted click on `tile`.
    Object.assign(window, {
        fakeBackend: fake,
        giveBomb: () => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBomb())
            return "💣 armed — press and hold on the planet"
        },
        giveBonus: (kind: Parameters<typeof fake.grantBonus>[0]) => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBonus(kind))
            return `${kind} running — click the planet`
        },
    })

    root.render(
        <StrictMode>
            <SignInGate callback={callback}>
                <App
                    ownershipsGetter={fake}
                    tileClicker={fake}
                    updatesListener={fake}
                    bonusListener={fake}
                    bomber={fake}
                    clickBudgetSource={fake}
                    chatBackend={new FakeChatBackend()}
                />
            </SignInGate>
        </StrictMode>,
    )
} else {
    const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
    const chatBackend = new ChatServiceBackend(newChatServiceClient(config))
    const account = new AccountStore(new ConnectAccountBackend(authClient), session, {
        navigate: (url) => window.location.assign(url),
        remember: rememberSignIn,
    })

    root.render(
        <StrictMode>
            <SignInGate callback={callback} account={account}>
                <App
                    ownershipsGetter={backend}
                    tileClicker={backend}
                    updatesListener={backend}
                    bonusListener={backend}
                    bomber={backend}
                    clickBudgetSource={backend}
                    chatBackend={chatBackend}
                    account={account}
                />
            </SignInGate>
        </StrictMode>,
    )
}
