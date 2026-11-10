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
import {chatterNameplateOf, FakeChatBackend} from "./backends/fakeChatBackend.ts"
import {FakeGifs} from "./backends/fakeGifs.ts"
import {GifSearch} from "./backends/gifs.ts"
import {KlipyGifs} from "./backends/klipyGifs.ts"
import {FakePresenceBackend} from "./backends/fakePresenceBackend.ts"
import {FakeBadges} from "./backends/fakeBadges.ts"
import {FakeNameplates} from "./backends/fakeNameplates.ts"
import {FakeMissionsBackend} from "./backends/fakeMissionsBackend.ts"
import {newScoringServiceClient} from "./backends/scoringBackend.ts"
import {closedOf} from "./domain/debrief.ts"
import {battleCommands, FakeTurnBackend, seasonRunningAt} from "./backends/fakeTurnBackend.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {ConnectionHealth, connectionInterceptor, followBrowser} from "./backends/connection.ts"
import {Restarts, restartInterceptor} from "./backends/restarts.ts"
import {FakeStandingsBackend} from "./backends/fakeStandingsBackend.ts"
import {ConnectStandingsBackend} from "./backends/standingsBackend.ts"
import {FakeSupportBackend} from "./backends/fakeSupportBackend.ts"
import {ConnectSupportBackend, newSupportServiceClient} from "./backends/supportBackend.ts"
import {loadPointGeometryData} from "./app/viewer/points.ts"
import type {Globe} from "./app/viewer/globe.ts"
import App from "./app/App.tsx"
import type {Backends} from "./app/play/PlayScreen.tsx"
import {ConnectAccountBackend} from "./backends/accountBackend.ts"
import {ConnectPlayerBackend, newKeepalivePlayerServiceClient, newPlayerServiceClient} from "./backends/playerBackend.ts"
import {ConnectProgressionBackend, newProgressionServiceClient} from "./backends/progressionBackend.ts"
import {PlayerCards} from "./backends/playerCards.ts"
import {ConnectMissionsBackend, newMissionServiceClient} from "./backends/missionsBackend.ts"
import {AccountStore} from "./app/account/accountStore.ts"
import {rememberSignIn} from "./app/account/rememberedSignIn.ts"
import SignInGate from "./app/account/SignInGate.tsx"
import {callbackOf, CALLBACK_PATH} from "./domain/signInCallback.ts"
import {ConnectClipsBackend, newClipsServiceClient} from "./backends/clipsBackend.ts"
import {FakeClipsBackend} from "./backends/fakeClipsBackend.ts"
import {FakePromoBackend} from "./backends/fakePromoBackend.ts"
import {ConnectPromoBackend, newPromoServiceClient} from "./backends/promoBackend.ts"
import {FakeAccountBackend} from "./backends/fakeAccountBackend.ts"
import {SHARE_FLAG_PARAM, sharedCountry, withoutSharedFlag} from "./domain/shareCard.ts"
import {linkedShot, withoutLinkedShot} from "./domain/linkedShot.ts"
import {discordJoinAsked, withoutDiscordJoin} from "./domain/discordJoin.ts"
import {enterFrontGuide} from "./app/frontGuide.ts"
import {useCountry} from "./app/countryStore.ts"
import {firstLanding, landingOf, SOURCE_TAG_PARAM, withoutLanding} from "./domain/landing.ts"
import {localLandingStore} from "./backends/landingStore.ts"
import {localLeaverStore} from "./backends/leaverStore.ts"
import {ON_CRAZYGAMES} from "./platform.ts"
import {connectCrazyGames} from "./backends/crazyGamesSdk.ts"
import {localAccountCarrier} from "./backends/accountCarrier.ts"
import {crazyGamesLanding} from "./domain/crazyGames.ts"
import {CrazyGamesPage} from "./backends/crazyGames.ts"
import {CRAZYGAMES_WORDS_URL, crazyGamesWords} from "./backends/crazyGamesWords.ts"
import {NO_FILTER, WordFilter} from "./domain/wordFilter.ts"
import {AlertsBackend} from "./backends/alerts.ts"
import {ConnectAlertsBackend, newAlertsServiceClient} from "./backends/alertsBackend.ts"
import {FakeAlertsBackend} from "./backends/fakeAlertsBackend.ts"
import {DiscordAlerts} from "./app/alerts/discordAlerts.ts"

// Strip the OAuth code and state from the URL before anything else can read or leak them.
const callback = callbackOf(new URL(window.location.href))
if (callback) window.history.replaceState(null, "", CALLBACK_PATH)

const page = new URL(window.location.href)
const shared = sharedCountry(page)
if (shared) useCountry.getState().choose(shared)
const shot = linkedShot(page)
const joinDiscord = discordJoinAsked(page)
const guided = enterFrontGuide()
const crazyGames = ON_CRAZYGAMES ? connectCrazyGames() : undefined
const words: Promise<WordFilter> = ON_CRAZYGAMES ? crazyGamesWords(CRAZYGAMES_WORDS_URL) : Promise.resolve(NO_FILTER)
const seen = landingOf(page, document.referrer)
const landing = crazyGames ? crazyGamesLanding(seen) : firstLanding(localLandingStore(), seen)
// CrazyGames' frame puts an invite's flag on this address as a plain ?f=, and its SDK reads it after this runs.
if (!crazyGames && (page.searchParams.has(SHARE_FLAG_PARAM) || page.searchParams.has(SOURCE_TAG_PARAM) || shot || joinDiscord)) {
    window.history.replaceState(null, "", withoutDiscordJoin(withoutLinkedShot(withoutLanding(withoutSharedFlag(page)))))
}

const connection = new ConnectionHealth()
followBrowser(connection, window)

const config = {
    baseUrl: API_BASE_URL,
    timeoutMs: 10_000,
    interceptors: [connectionInterceptor(connection), restartInterceptor(new Restarts(() => window.location.reload()))],
}

const sitekey = import.meta.env.VITE_TURNSTILE_SITEKEY
const klipyKey: string | undefined = import.meta.env.VITE_KLIPY_KEY
const authClient = newAuthServiceClient(config, crazyGames && localAccountCarrier())
const attest = sitekey ? turnstileAttester(sitekey, "session") : async () => ""
const session: SessionProvider = sitekey
    ? new SessionClient(authClient, attest, {store: localTokenStore(), landing})
    : new NoSession()

const root = createRoot(document.getElementById('root')!)

// Planet hears player's PlayerCreated over Redis, and player's outbox sends once a second.
const FAKE_GRANT_LANDS_MS = 1_200
// Alerts hears auth's DiscordJoinGranted over Redis, auth's outbox sends once a second, then alerts asks Discord.
const FAKE_JOIN_LANDS_MS = 1_500

// Discord's sign-in leaves for Discord and comes back on clickplanet.lol, out of CrazyGames' frame and its carried
// session: and CrazyGames allows no sign-in but its own.
function discordAlertsOf(backend: AlertsBackend, account: AccountStore): DiscordAlerts | undefined {
    if (crazyGames) return undefined
    return new DiscordAlerts(backend, () => account.connectDiscord(), joinDiscord)
}

// CrazyGames' word list cannot read the words in a GIF.
function gifsOf(search: GifSearch | undefined): GifSearch | undefined {
    return crazyGames ? undefined : search
}

function followCrazyGames(page: CrazyGamesPage | undefined, account: AccountStore) {
    if (!page) return
    page.user.listenForSignIn(() => void account.followCrazyGames())
    void account.followCrazyGames()
}

function start(backends: Backends) {
    void words.then((filter) => root.render(
        <StrictMode>
            <SignInGate callback={callback} account={backends.account}>
                <App linkedShot={shot}
                     showFront={guided}
                     backends={backends}
                     words={filter}/>
            </SignInGate>
        </StrictMode>,
    ))
}

if (import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND) {
    const borders = loadBorders(BORDERS_URL)
    const fake = new FakeBackend(100, {
        tilePositions: () => loadPointGeometryData().then((data) => data.positions),
        landmasses: () => borders.then(({assignment, codes}) => ({assignment, count: codes.length})),
    })
    const fakePresence = new FakePresenceBackend()
    const fakeAccount = new FakeAccountBackend(fakePresence)
    const account = new AccountStore(fakeAccount, fakeAccount, fakeAccount, new NoSession(), {
        navigate: (url) => window.location.assign(url),
        remember: rememberSignIn,
        claimFlag: () => fake.claimFlag(),
        crazyGames: crazyGames?.user,
        leavers: crazyGames && localLeaverStore(),
    })
    followCrazyGames(crazyGames, account)
    const alerts = discordAlertsOf(
        new FakeAlertsBackend(async () => (await fakeAccount.me()).linked.length > 0, FAKE_JOIN_LANDS_MS), account)
    const fakeClips = new FakeClipsBackend()
    const fakePromo = new FakePromoBackend()
    const fakeMissions = new FakeMissionsBackend((kind) => {
        if (kind === "bomb") fake.grantBomb()
        else fake.grantBonus(kind)
    }, fake)
    const endDayAt = (endedAt: number) => {
        const today = fakeTurn.today()
        if (!today) return "no day runs"
        const round = fakeStandings.closeRound(today.roundId)
        const debrief = fakeTurn.endDay(endedAt)
        if (debrief) fakeChat.announceRound(closedOf(debrief, round))
        return `day ${today.number} is over`
    }
    // As the gamemaster: a battle ends the day at its start, and scoring's debrief of it comes after the turn.
    const endDayAfterTurn = (since: number) => setTimeout(() => endDayAt(since), 1000)
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
        unlimitEnclosures: (on = true) => {
            fake.unlimitEnclosures(on)
            return on ? "enclosures are unlimited: enclose is on, and spends nothing" : "enclosures are spent again"
        },
        lockFlags: () => {
            fake.lockFlags(true)
            return "the flags are locked: no change of country"
        },
        openFlags: () => {
            fake.lockFlags(false)
            return "the flags may change again"
        },
        giveBomb: () => {
            const globe = (window as {clickplanetGlobe?: Globe}).clickplanetGlobe
            if (!globe) return "the globe is not loaded yet"
            globe.takeReward(fake.grantBomb())
            return "💣 in your inventory — aim it from there"
        },
        giveBox: () => fake.offerBox()
            ? "a box is flying round the globe — catch it"
            : "every stack is full: no box flies",
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
        giveTitle: (id?: string) => fakePresence.earnTitle(id)
            ? "a title is unlocked"
            : "no page listens: a guest earns no title, signIn() first",
        finishMissions: () => {
            fakeMissions.prime()
            return "every mission is one tile from done: take a tile for your flag to finish and get them"
        },
        closeDay: () => endDayAt(Date.now()),
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
            endDayAfterTurn(since)
            return `a battle is on, at ×${times}`
        },
        startFinale: (description?: string, times = 3) => {
            const since = Date.now()
            const roundId = fakeTurn.startFinale(since, description, times)
            fakeStandings.openBattle(roundId, since, times)
            endDayAfterTurn(since)
            return `the final battle is on, at ×${times}`
        },
        endBattle: () => {
            fakeTurn.endBattle()
            return "the battle is over: closeBattle() or closeFinale() sends its results"
        },
        offerSignIn: () => {
            fakeAccount.offerSignIn()
            void account.load().then(() => account.followCrazyGames())
            return "a guest is offered to sign in: run out of clicks, then signIn() finishes a sign-in"
        },
        signIn: (name = "Nova") => {
            void account.completeSignIn(name, "").then(() => {
                const state = account.state()
                if (state.kind === "ready" && state.guestLinked) setTimeout(() => fake.grantBonus("refill"), FAKE_GRANT_LANDS_MS)
            })
            return `signed in as ${name}: sign out from your profile (You, at the foot of the rail)`
        },
        giveClip: (headline?: string) => {
            const clip = fakeClips.giveClip(headline)
            return `"${clip.title}" names you: signIn() reads it now, or come back to this tab 10 minutes after the last read`
        },
    })

    const fakeChat = new FakeChatBackend()
    const fakeSupport = new FakeSupportBackend()
    Object.assign(window, {fakeChat, fakeSupport, fakePromo})
    const fakeTurn = new FakeTurnBackend(seasonRunningAt(Date.now()))
    Object.assign(window, battleCommands(fakeTurn))
    const fakeStandings = new FakeStandingsBackend()
    const fakeBadges = new FakeBadges([fakePresence, fakeStandings])
    const fakeNameplates = new FakeNameplates([fakeAccount, fakePresence, {nameplateOf: chatterNameplateOf}, fakeStandings])
    Object.assign(window, {fakeBadges, fakeNameplates})
    const clicker = fakeMissions.counting(fakeStandings.counting(fake))
    fake.listenForBombs((drop) => fakeChat.announceBomb(drop))
    fakeMissions.listenForFlagMissionsDone((flag, mission) => fakeChat.announceFlagMission(flag, mission))
    fake.listenForFortifications((fortification) => void borders.then(({codes}) => fakeChat.announceFortify({
        ...fortification, ground: codes[fortification.landmass], tiles: fake.landmassTiles(fortification.landmass),
    })))

    start({
        ownershipsGetter: fake,
        tileClicker: clicker,
        updatesListener: fake,
        bonusListener: fake,
        quizMaster: fake,
        bomber: fake,
        refiller: fake,
        shielder: fake,
        allegiance: fake,
        clickBudgetSource: fake,
        connection,
        chatBackend: fakeChat,
        presence: fakePresence,
        playerInfo: new PlayerCards(fakePresence, fakePresence),
        badges: fakeBadges,
        nameplates: fakeNameplates,
        titles: fakePresence,
        tileHolders: fake,
        landmasses: fake,
        turn: fakeTurn,
        standings: fakeStandings,
        support: fakeSupport,
        missions: fakeMissions,
        chargesRefresher: fake,
        account,
        clips: fakeClips,
        promo: fakePromo,
        crazyGames,
        alerts,
        gifs: gifsOf(new FakeGifs()),
    })
} else {
    const backend = new PlanetBackend(newClickServiceClient(config), 100, session)
    const chatBackend = new ChatServiceBackend(newChatServiceClient(config), session, newKeepaliveChatServiceClient(config))
    const turn = new ConnectTurnBackend(newGamemasterServiceClient(config))
    const player = new ConnectPlayerBackend(newPlayerServiceClient(config), session, newKeepalivePlayerServiceClient(config))
    const progression = new ConnectProgressionBackend(newProgressionServiceClient(config), session)
    const standings = new ConnectStandingsBackend(newScoringServiceClient(config), session)
    const missions = new ConnectMissionsBackend(newMissionServiceClient(config), session)
    const support = new ConnectSupportBackend(newSupportServiceClient(config), session)
    const clips = new ConnectClipsBackend(newClipsServiceClient(config), session)
    const promo = new ConnectPromoBackend(newPromoServiceClient(config))
    const account = new AccountStore(new ConnectAccountBackend(authClient, attest, landing), player, progression, session, {
        navigate: (url) => window.location.assign(url),
        remember: rememberSignIn,
        claimFlag: () => backend.claimFlag(),
        crazyGames: crazyGames?.user,
        leavers: crazyGames && localLeaverStore(),
    })
    followCrazyGames(crazyGames, account)
    const alerts = discordAlertsOf(new ConnectAlertsBackend(newAlertsServiceClient(config), session), account)

    start({
        ownershipsGetter: backend,
        tileClicker: backend,
        updatesListener: backend,
        bonusListener: backend,
        quizMaster: backend,
        bomber: backend,
        refiller: backend,
        shielder: backend,
        allegiance: backend,
        clickBudgetSource: backend,
        connection,
        chatBackend,
        account,
        presence: player,
        playerInfo: new PlayerCards(player, progression),
        badges: progression,
        nameplates: player,
        titles: progression,
        tileHolders: backend,
        landmasses: backend,
        turn,
        standings,
        support,
        missions,
        chargesRefresher: backend,
        clips,
        promo,
        crazyGames,
        alerts,
        gifs: gifsOf(klipyKey ? new KlipyGifs(klipyKey) : undefined),
    })
}
