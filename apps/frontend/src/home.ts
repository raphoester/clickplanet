import {nextBattleShown} from "./app/home/battle.ts"
import {finaleShown} from "./app/home/finale.ts"
import {seasonShown} from "./app/home/season.ts"
import {carryQuery} from "./app/home/playLinks.ts"
import {firstLanding, landingOf} from "./domain/landing.ts"
import {localLandingStore} from "./backends/landingStore.ts"
import {battleCommands, FakeTurnBackend, seasonRunningAt} from "./backends/fakeTurnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"
import {speak} from "./i18n/page.ts"
import {localeOf} from "./i18n/load.ts"
import {ENGLISH, Translator} from "./i18n/translator.ts"

// The home page is English until it has its own languages, in the browser's own English for its dates.
speak(new Translator("en", ENGLISH, localeOf("en", navigator.languages ?? [navigator.language])))

const fake = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
const turns: TurnBackend = fake
    ? fakeTurns()
    : new ConnectTurnBackend(newGamemasterServiceClient({baseUrl: API_BASE_URL}))

// Here, not on /play: the game's page sees this page as where the player came from.
void firstLanding(localLandingStore(), landingOf(new URL(location.href), document.referrer))
carryQuery(document, location.search)

const season = seasonShown(document)
const battle = nextBattleShown(document)
const finale = finaleShown(document)
turns.follow({
    onTurn: finale.showTurn,
    onBattles: (battles) => {
        battle.show(battles)
        finale.showBattles(battles)
    },
    onSeasons: ({current}) => season.show(current),
    onDebrief: () => {},
})

function fakeTurns(): FakeTurnBackend {
    const turns = new FakeTurnBackend(seasonRunningAt(Date.now()))
    Object.assign(window, battleCommands(turns))
    return turns
}
