import {createRoot} from 'react-dom/client'
import {StrictMode} from "react"
import './tokens.css'
import './index.css'

import {newClickServiceClient, PlanetBackend} from "./backends/planetBackend.ts"
import {NoSession, SessionProvider} from "./backends/session.ts"
import {localTokenStore, newAuthServiceClient, SessionClient, turnstileAttester} from "./backends/turnstileSession.ts"
import {ChatServiceBackend, newChatServiceClient, newKeepaliveChatServiceClient} from "./backends/chatBackend.ts"
import {FakeBackend} from "./backends/fakeBackend.ts"
import {loadBorders} from "./app/viewer/borderField.ts"
import {BORDERS_URL} from "./app/viewer/bordersAsset.ts"
import {FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {FakePresenceBackend} from "./backends/fakePresenceBackend.ts"
import {FakeMissionsBackend} from "./backends/fakeMissionsBackend.ts"
import {newScoringServiceClient} from "./backends/scoringBackend.ts"
import {closedOf} from "./domain/debrief.ts"
import {battleCommands, FakeTurnBackend} from "./backends/fakeTurnBackend.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {ConnectionHealth, connectionInterceptor, followBrowser} from "./backends/connection.ts"
import {FakeStandingsBackend} from "./backends/fakeStandingsBackend.ts"
import {ConnectStandingsBackend} from "./backends/standingsBackend.ts"
import {FakeSupportBackend} from "./backends/fakeSupportBackend.ts"
import {ConnectSupportBackend, newSupportServiceClient} from "./backends/supportBackend.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import type {Globe} from "./app/viewer/globe.ts"
import App from "./app/App.tsx"
import {ConnectAccountBackend} from "./backends/accountBackend.ts"
import {ConnectPlayerBackend, newKeepalivePlayerServiceClient, newPlayerServiceClient} from "./backends/playerBackend.ts"
import {ConnectMissionsBackend, newMissionServiceClient} from "./backends/missionsBackend.ts"
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

const connection = new ConnectionHealth()
followBrowser(connection, window)

const config = {
    baseUrl: API_BASE_URL,
    timeoutMs: 10_000,
    interceptors: [connectionInterceptor(connection)],
}

const sitekey = import.meta.env.VITE_TURNSTILE_SITEKEY
const authClient = newAuthServiceClient(config)
const attest = sitekey ? turnstileAttester(sitekey, "session") : async () => ""
const session: SessionProvider = sitekey
    ? new SessionClient(authClient, attest, {store: localTokenStore()})
    : new NoSession()

const root = createRoot(document.getElementById('root')!)

if (import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND) {
    const borders = loadBorders(BORDERS_URL)
    const fake = new FakeBackend(100, {
        tilePositions: () => loadPointGeometryData().then((data) => data.positions),
        landmasses: () => borders.then(({assignment, codes}) => ({assignment, count: codes.length})),
    })
    const fakePresence = new FakePresenceBackend()
    const fakeMissions = new FakeMissionsBackend((kind) => {
        if (kind === "bomb") fake.grantBomb()
        else fake.grantBonus(kind)
    }, fake)
    Object.assign(window, {
        fakeBackend: fake,
        connection,
        freezeMap: () => {
            fake.freeze(true)
            return "the map is frozen: every click is refused"
        },
        thawMap: () => {
            fake.freeze(false)
            return "the map takes play again"
        },
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
        finishMissions: () => {
            fakeMissions.prime()
            return "every mission is one tile from done: take a tile for your flag to finish and get them"
        },
        closeDay: () => {
            const today = fakeTurn.today()
            if (!today) return "no day runs"
            const round = fakeStandings.closeRound(today.roundId)
            const debrief = fakeTurn.endDay()
            if (debrief) fakeChat.announceRound(closedOf(debrief, round))
            return `day ${today.number} is over`
        },
        closeBattle: () => {
            const round = fakeStandings.closeBattle()
            const debrief = fakeTurn.endBattleRound(round.roundId, round.times, round.endedAt, false)
            fakeChat.announceRound(closedOf(debrief, round))
            return `battle ${debrief.number} is closed`
        },
        closeFinale: () => {
            const round = fakeStandings.closeBattle()
            const battle = fakeTurn.endBattleRound(round.roundId, round.times, round.endedAt, true)
            fakeChat.announceRound(closedOf(battle, round))
            const season = fakeTurn.endSeason(round.roundId, battle)
            if (season) fakeChat.announceChampion(closedOf(season, round))
            return "the Final Battle is closed, and the season with it"
        },
        startBattle: (minutes = 120, description?: string, times = 3) => {
            const since = Date.now()
            const endsAt = since + minutes * 60_000
            const entryId = crypto.randomUUID()
            fakeStandings.openBattle(entryId, since, times)
            fakeTurn.startBattle(entryId, since, endsAt, description, times)
            return `a battle is on, at ×${times}`
        },
        startFinale: (description?: string, times = 3) => {
            const since = Date.now()
            const roundId = fakeTurn.startFinale(since, description, times)
            fakeStandings.openBattle(roundId, since, times)
            return `the final battle is on, at ×${times}`
        },
        endBattle: () => {
            fakeTurn.endBattle()
            return "the battle is over: closeBattle() or closeFinale() sends its results"
        },
    })

    const fakeChat = new FakeChatBackend()
    const fakeSupport = new FakeSupportBackend()
    Object.assign(window, {fakeChat, fakeSupport})
    const fakeTurn = new FakeTurnBackend()
    Object.assign(window, battleCommands(fakeTurn))
    const fakeStandings = new FakeStandingsBackend()
    const clicker = fakeMissions.counting(fakeStandings.counting(fake))
    fake.listenForBombs((drop) => fakeChat.announceBomb(drop))
    fake.listenForFortifications((fortification) => void borders.then(({codes}) => fakeChat.announceFortify({
        ...fortification, ground: codes[fortification.landmass], tiles: fake.landmassTiles(fortification.landmass),
    })))

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
                    allegiance={fake}
                    clickBudgetSource={fake}
                    connection={connection}
                    chatBackend={fakeChat}
                    presence={fakePresence}
                    playerInfo={fakePresence}
                    tileHolders={fake}
                    landmasses={fake}
                    turn={fakeTurn}
                    standings={fakeStandings}
                    support={fakeSupport}
                    missions={fakeMissions}
                    chargesRefresher={fake}
                />
            </SignInGate>
        </StrictMode>,
    )
} else {
    const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
    const chatBackend = new ChatServiceBackend(newChatServiceClient(config), session, newKeepaliveChatServiceClient(config))
    const turn = new ConnectTurnBackend(newGamemasterServiceClient(config))
    const player = new ConnectPlayerBackend(newPlayerServiceClient(config), session, newKeepalivePlayerServiceClient(config))
    const standings = new ConnectStandingsBackend(newScoringServiceClient(config), session)
    const missions = new ConnectMissionsBackend(newMissionServiceClient(config), session)
    const support = new ConnectSupportBackend(newSupportServiceClient(config), session)
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
                    allegiance={backend}
                    clickBudgetSource={backend}
                    connection={connection}
                    chatBackend={chatBackend}
                    account={account}
                    presence={player}
                    playerInfo={player}
                    tileHolders={backend}
                    landmasses={backend}
                    turn={turn}
                    standings={standings}
                    support={support}
                    missions={missions}
                    chargesRefresher={backend}
                />
            </SignInGate>
        </StrictMode>,
    )
}
