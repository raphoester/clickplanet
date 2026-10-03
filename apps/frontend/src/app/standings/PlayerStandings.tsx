import {MySeason, Standing, StandingsBackend} from "../../backends/standings.ts"
import {NameColor} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import RankCoin from "../components/RankCoin.tsx"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {Caller, useMySeason} from "./useMySeason.ts"
import {useStandings} from "./useStandings.ts"
import YourSeason from "./YourSeason.tsx"
import "./Standings.css"

export type PlayerStandingsProps = {
    backend: StandingsBackend
    countryCode: string
    label: string
    caller: Caller
    listenForClicks: ListenForClicks
    onSignIn?: () => void
}

export default function PlayerStandings(props: PlayerStandingsProps) {
    const standings = useStandings(props.backend, props.countryCode)
    const mine = useMySeason(props.backend, props.caller, props.listenForClicks)
    const name = props.caller.username
    const listed = standings?.some((standing) => standing.name === name) ?? false

    return <>
        <YourSeason mine={mine} caller={props.caller} onSignIn={props.onSignIn}/>
        {standings && <StandingsTable label={props.label}
                                      standings={standings}
                                      name={name}
                                      own={listed ? undefined : ownLine(props.caller, mine, props.countryCode)}/>}
    </>
}

function ownLine(caller: Caller, mine: MySeason | undefined, countryCode: string): Standing | undefined {
    const rank = countryCode === "" ? mine?.globalRank : mine?.countryCode === countryCode ? mine.countryRank : undefined
    if (caller.username === undefined || mine?.countryCode === undefined || rank === undefined) return undefined

    return {
        rank,
        name: caller.username,
        color: caller.color ?? NameColor.UNSPECIFIED,
        countryCode: mine.countryCode,
        tiles: mine.tiles,
    }
}

type StandingsTableProps = {
    label: string
    standings: readonly Standing[]
    name?: string
    own?: Standing
}

function StandingsTable({label, standings, name, own}: StandingsTableProps) {
    if (standings.length === 0 && !own) return <p className="standings-empty">Nobody yet.</p>

    return <table className="leaderboard-table standings-table" aria-label={label}>
        <thead>
        <tr>
            <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
            <th className="leaderboard-table-head" scope="col">Player</th>
            <th className="leaderboard-table-head leaderboard-table-number" scope="col">Tiles</th>
        </tr>
        </thead>
        <tbody>
        {standings.map((standing) => <StandingRow key={standing.name}
                                                  standing={standing}
                                                  you={name !== undefined && standing.name === name}/>)}
        {own && <>
            <tr className="standings-gap" aria-hidden="true">
                <td colSpan={3}/>
            </tr>
            <StandingRow standing={own} you/>
        </>}
        </tbody>
    </table>
}

function StandingRow({standing, you}: {standing: Standing, you: boolean}) {
    const country = Countries.get(standing.countryCode)

    return <tr className={you ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
               aria-current={you ? "true" : undefined}>
        <td className="leaderboard-entry-index"><RankCoin rank={standing.rank} you={you}/></td>
        <td className="standings-player">
            {country && <span className="standings-flag" role="img" aria-label={country.name} title={country.name}>
                <CountryFlag code={country.code}/>
            </span>}
            <span className="standings-name"
                  style={authorStyle({name: standing.name, color: standing.color, guest: false})}
                  title={standing.name}>
                {standing.name}
            </span>
        </td>
        <td className="leaderboard-table-number">{standing.tiles}</td>
    </tr>
}
