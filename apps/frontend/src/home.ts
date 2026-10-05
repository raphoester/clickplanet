import {showFinale} from "./app/home/finale.ts"
import {carryQuery} from "./app/home/playLinks.ts"
import {FakeSeasonBackend, SEASON_ZERO} from "./backends/fakeSeasonBackend.ts"
import {SeasonBackend} from "./backends/season.ts"
import {ConnectSeasonBackend, newSeasonServiceClient} from "./backends/seasonBackend.ts"
import {API_BASE_URL} from "./backends/transport.ts"

const seasons: SeasonBackend = import.meta.env.DEV && import.meta.env.VITE_FAKE_BACKEND
    ? new FakeSeasonBackend(SEASON_ZERO)
    : new ConnectSeasonBackend(newSeasonServiceClient({baseUrl: API_BASE_URL}))

carryQuery(document, location.search)

seasons.season().then(
    (season) => season && showFinale(document, season),
    (e) => console.error("Could not read the season", e),
)
