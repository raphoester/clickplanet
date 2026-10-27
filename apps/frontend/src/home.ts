import {showChampion} from "./app/home/champion.ts"
import {showSeason} from "./app/home/season.ts"
import {carryQuery} from "./app/home/playLinks.ts"
import {FakeHallOfFameBackend, sampleEndedSeason} from "./backends/fakeHallOfFameBackend.ts"
import {FakeSeasonBackend, SEASON_ZERO} from "./backends/fakeSeasonBackend.ts"
import {HallOfFameBackend} from "./backends/hallOfFame.ts"
import {ConnectHallOfFameBackend} from "./backends/hallOfFameBackend.ts"
import {SeasonBackend} from "./backends/season.ts"
import {ConnectSeasonBackend, newSeasonServiceClient} from "./backends/seasonBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"

const client = newSeasonServiceClient({baseUrl: API_BASE_URL})
const seasons: SeasonBackend = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
    ? new FakeSeasonBackend(SEASON_ZERO)
    : new ConnectSeasonBackend(client)
const hallOfFame: HallOfFameBackend = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
    ? new FakeHallOfFameBackend([sampleEndedSeason()])
    : new ConnectHallOfFameBackend(client)

carryQuery(document, location.search)

seasons.season().then(
    (season) => season && showSeason(document, season),
    (e) => console.error("Could not read the season", e),
)

hallOfFame.hallOfFame().then(
    (hall) => showChampion(document, hall),
    (e) => console.error("Could not read the Hall of Fame", e),
)
