import {Round, StandingsBackend} from "../../backends/standings.ts"
import {Countries, Country} from "../../domain/countries.ts"
import {RaceRow, raceTable} from "../../domain/race.ts"
import {leftUntil} from "../../domain/seasonClock.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import RankCoin from "../components/RankCoin.tsx"
import {StatTile, StatTiles} from "../components/StatTiles.tsx"
import {useNow} from "../season/useNow.ts"
import {truncate} from "../truncate.ts"
import {useRace} from "./useRace.ts"
import "./Standings.css"

const NAME_MAX_LENGTH = 18

export type RaceStandingsProps = {
    backend: StandingsBackend
    country: Country
}

export default function RaceStandings({backend, country}: RaceStandingsProps) {
    const race = useRace(backend)
    if (!race) return null

    const table = raceTable(race, country.code)

    return <>
        {race.round && <Today round={race.round} country={country}/>}
        {table.listed.length === 0
            ? <p className="standings-empty">Nobody yet.</p>
            : <table className="leaderboard-table standings-table" aria-label="Season">
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number race-points" scope="col">Points</th>
                </tr>
                </thead>
                <tbody>
                {table.listed.map((row) => <RaceEntry key={row.countryCode} row={row} you={row.countryCode === country.code}/>)}
                {table.below && <>
                    <tr className="standings-gap" aria-hidden="true">
                        <td colSpan={3}/>
                    </tr>
                    <RaceEntry row={table.below} you/>
                </>}
                </tbody>
            </table>}
    </>
}

function Today({round, country}: {round: Round, country: Country}) {
    const now = useNow()
    const standing = round.standings.find((line) => line.countryCode === country.code)

    return <section className="your-season" aria-label="Today">
        <StatTiles>
            <StatTile label={round.finale ? "Final Battle" : `Day ${round.number}`} value={leftUntil(round.endsAt, now)}/>
            <StatTile label={country.name} value={standing ? `#${standing.rank}` : "—"}/>
        </StatTiles>
    </section>
}

function RaceEntry({row, you}: {row: RaceRow, you: boolean}) {
    const country = Countries.get(row.countryCode)

    return <tr className={you ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
               aria-current={you ? "true" : undefined}>
        <td className="leaderboard-entry-index"><RankCoin rank={row.rank} you={you}/></td>
        <td className="leaderboard-entry-country">
            <CountryFlag code={row.countryCode}/>
            {truncate(country?.name ?? row.countryCode, NAME_MAX_LENGTH)}
        </td>
        <td className="leaderboard-table-number race-points">
            {row.points}
            {row.today > 0 && <span className="race-today">+{row.today}<span className="sr-only"> today</span></span>}
        </td>
    </tr>
}
