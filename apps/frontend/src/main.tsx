import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './tokens.css'
import './index.css'

import {newClickServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {localTokenStore, newAuthServiceClient, SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient, newKeepaliveChatServiceClient} from "./backends/chatBackend.ts"
import {FakeBackend} from "./backends/fakeBackend.ts"
import {FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {FakePresenceBackend} from "./backends/fakePresenceBackend.ts"
import {FakeSeasonBackend, SEASON_ZERO} from "./backends/fakeSeasonBackend.ts"
import {ConnectSeasonBackend, newSeasonServiceClient} from "./backends/seasonBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {FakeStandingsBackend} from "./backends/fakeStandingsBackend.ts"
import {ConnectStandingsBackend} from "./backends/standingsBackend.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import type {Globe} from "./app/viewer/globe.ts"
import App from "./app/App.tsx"
import {ConnectAccountBackend} from "./backends/accountBackend.ts"
import {ConnectPlayerBackend, newKeepalivePlayerServiceClient, newPlayerServiceClient} from "./backends/playerBackend.ts"
import {AccountStore} from "./app/account/accountStore.ts"
import {rememberSignIn} from "./app/account/rememberedSignIn.ts"
import SignInGate from "./app/account/SignInGate.tsx"
import {callbackOf, CALLBACK_PATH} from "./domain/signInCallback.ts"
import {SHARE_FLAG_PARAM, sharedCountry, withoutSharedFlag} from "./domain/shareCard.ts"

// Strip the OAuth code and state from the URL before anything else can read or leak them.
const callback = callbackOf(new URL(window.location.href))
if (callback) window.history.replaceState(null, "", CALLBACK_PATH)

const page = new URL(window.location.href)
const shared = sharedCountry(page)
if (page.searchParams.has(SHARE_FLAG_PARAM)) window.history.replaceState(null, "", withoutSharedFlag(page))

const config = {
    baseUrl: API_BASE_URL,
    timeoutMs: 2000,
}

const sitekey = import.meta.env.VITE_TURNSTILE_SITEKEY
const authClient = newAuthServiceClient(config)
const attest = sitekey ? turnstileAttester(sitekey, "session") : async () => ""
const session: SessionProvider = sitekey
    ? new SessionClient(authClient, attest, {store: localTokenStore()})
    : new NoSession()

const root = createRoot(document.getElementById('root')!)

if (import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND) {
    const fake = new FakeBackend(100, {
        tilePositions: () => loadPointGeometryData().then((data) => data.positions),
    })
    const fakePresence = new FakePresenceBackend()
    const fakeChat = new FakeChatBackend()
    Object.assign(window, {
        fakeBackend: fake,
        giveBomb: () => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBomb())
            return "💣 in your inventory — aim it from there"
        },
        giveQuiz: () => {
            fake.offerQuiz()
            return "a quiz is up — press it, then you have 8 seconds"
        },
        giveBonus: (kind: Parameters<typeof fake.grantBonus>[0]) => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBonus(kind))
            return `${kind} in your inventory — switch it on from there`
        },
        giveTitle: (id?: string) => {
            fakePresence.earnTitle(id)
            return "a title is unlocked"
        },
        announceLead: (leader: string, passed: string, season?: number) => {
            fakeChat.announceLead(leader, passed, season)
            return "a lead change is in the chat"
        },
        announceWinner: (winner: string, season?: number) => {
            fakeChat.announceWinner(winner, season)
            return "a winner is in the chat"
        },
    })

    Object.assign(window, {fakeChat})
    const fakeSeason = new FakeSeasonBackend(SEASON_ZERO)
    const fakeStandings = new FakeStandingsBackend()
    const clicker = fakeStandings.counting(fake)
    fake.listenForBombs((drop) => fakeChat.announceBomb(drop))

    root.render(
        <StrictMode>
            <SignInGate callback={callback}>
                <App
                    sharedCountry={shared}
                    ownershipsGetter={fake}
                    tileClicker={clicker}
                    updatesListener={fake}
                    bonusListener={fake}
                    quizMaster={fake}
                    bomber={fake}
                    refiller={fake}
                    shielder={fake}
                    clickBudgetSource={fake}
                    chatBackend={fakeChat}
                    presence={fakePresence}
                    playerInfo={fakePresence}
                    season={fakeSeason}
                    standings={fakeStandings}
                />
            </SignInGate>
        </StrictMode>,
    )
} else {
    const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
    const chatBackend = new ChatServiceBackend(newChatServiceClient(config), session, newKeepaliveChatServiceClient(config))
    const season = new ConnectSeasonBackend(newSeasonServiceClient(config))
    const player = new ConnectPlayerBackend(newPlayerServiceClient(config), session, newKeepalivePlayerServiceClient(config))
    const standings = new ConnectStandingsBackend(newSeasonServiceClient(config), session)
    const account = new AccountStore(new ConnectAccountBackend(authClient, attest), player, session, {
        navigate: (url) => window.location.assign(url),
        remember: rememberSignIn,
    })

    root.render(
        <StrictMode>
            <SignInGate callback={callback} account={account}>
                <App
                    sharedCountry={shared}
                    ownershipsGetter={backend}
                    tileClicker={backend}
                    updatesListener={backend}
                    bonusListener={backend}
                    quizMaster={backend}
                    bomber={backend}
                    refiller={backend}
                    shielder={backend}
                    clickBudgetSource={backend}
                    chatBackend={chatBackend}
                    account={account}
                    presence={player}
                    playerInfo={player}
                    season={season}
                    standings={standings}
                />
            </SignInGate>
        </StrictMode>,
    )
}
