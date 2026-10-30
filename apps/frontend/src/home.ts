import {nextBattleShown} from "./app/home/battle.ts"
import {seasonShown} from "./app/home/season.ts"
import {carryQuery} from "./app/home/playLinks.ts"
import {firstLanding, landingOf} from "./domain/landing.ts"
import {localLandingStore} from "./backends/landingStore.ts"
import {battleCommands, FakeTurnBackend} from "./backends/fakeTurnBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"

const fake = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
const turns: TurnBackend = fake
    ? fakeTurns()
    : new ConnectTurnBackend(newGamemasterServiceClient({baseUrl: API_BASE_URL}))

// Here, not on /play: the game's page sees this page as where the player came from.
firstLanding(localLandingStore(), landingOf(new URL(location.href), document.referrer))
carryQuery(document, location.search)

const season = seasonShown(document)
const battle = nextBattleShown(document)
turns.follow({onTurn: () => {}, onBattles: battle.show, onSeasons: ({current}) => season.show(current), onDebrief: () => {}})

function fakeTurns(): FakeTurnBackend {
    const turns = new FakeTurnBackend()
    Object.assign(window, battleCommands(turns))
    return turns
}
