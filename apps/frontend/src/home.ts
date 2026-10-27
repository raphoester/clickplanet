import {showChampion} from "./app/home/champion.ts"
import {showNextBattle} from "./app/home/battle.ts"
import {showSeason} from "./app/home/season.ts"
import {carryQuery} from "./app/home/playLinks.ts"
import {FakeHallOfFameBackend, sampleEndedSeason} from "./backends/fakeHallOfFameBackend.ts"
import {FakeSeasonBackend, SEASON_ZERO} from "./backends/fakeSeasonBackend.ts"
import {HallOfFameBackend} from "./backends/hallOfFame.ts"
import {ConnectHallOfFameBackend} from "./backends/hallOfFameBackend.ts"
import {battleCommands, FakeTurnBackend} from "./backends/fakeTurnBackend.ts"
import {SeasonBackend} from "./backends/season.ts"
import {ConnectSeasonBackend, newSeasonServiceClient} from "./backends/seasonBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"
import {TurnBackend} from "./backends/turn.ts"
import {ConnectTurnBackend, newGamemasterServiceClient} from "./backends/turnBackend.ts"

const fake = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
const client = newSeasonServiceClient({baseUrl: API_BASE_URL})
const seasons: SeasonBackend = fake
    ? new FakeSeasonBackend(SEASON_ZERO)
    : new ConnectSeasonBackend(client)
const hallOfFame: HallOfFameBackend = fake
    ? new FakeHallOfFameBackend([sampleEndedSeason()])
    : new ConnectHallOfFameBackend(client)
const turns: TurnBackend = fake
    ? fakeTurns()
    : new ConnectTurnBackend(newGamemasterServiceClient({baseUrl: API_BASE_URL}))

carryQuery(document, location.search)

seasons.season().then(
    (season) => season && showSeason(document, season),
    (e) => console.error("Could not read the season", e),
)

hallOfFame.hallOfFame().then(
    (hall) => showChampion(document, hall),
    (e) => console.error("Could not read the Hall of Fame", e),
)

showNextBattle(document, turns)

function fakeTurns(): FakeTurnBackend {
    const turns = new FakeTurnBackend()
    Object.assign(window, battleCommands(turns))
    return turns
}
