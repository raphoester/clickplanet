import {useId} from "react"
import {EndedSeason} from "../../backends/hallOfFame.ts"
import {PlayerLine} from "../../backends/player.ts"
import {Score} from "../../backends/standings.ts"
import {Countries} from "../../domain/countries.ts"
import {championsOf, finalTableOf} from "../../domain/hallOfFame.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Modal from "../components/Modal.tsx"
import RankCoin from "../components/RankCoin.tsx"
import {Trophy} from "../season/SeasonFacts.tsx"
import {StandingsTable} from "../standings/PlayerStandings.tsx"
import {truncate} from "../truncate.ts"
import "../Leaderboard.css"
import "../season/Season.css"
import "./HallOfFame.css"

const NAME_MAX_LENGTH = 18

export type HallOfFameProps = {
    hall: readonly EndedSeason[]
    countryCode: string
    accountId?: string
    onOpenPlayer?: (player: PlayerLine) => void
    onClose: () => void
}

export default function HallOfFame({hall, countryCode, accountId, onOpenPlayer, onClose}: HallOfFameProps) {
    return <Modal title="Hall of Fame" className="hall-of-fame" onClose={onClose}>
        {hall.map((season) => <SeasonFame key={season.number}
                                          season={season}
                                          countryCode={countryCode}
                                          accountId={accountId}
                                          onOpenPlayer={onOpenPlayer}/>)}
    </Modal>
}

type SeasonFameProps = Omit<HallOfFameProps, "hall" | "onClose"> & {season: EndedSeason}

function SeasonFame({season, countryCode, accountId, onOpenPlayer}: SeasonFameProps) {
    const label = useId()
    const {listed, own} = finalTableOf(season, countryCode)

    return <section className="hall-season" aria-labelledby={label}>
        <div className="panel-box hall-champion">
            <span className="hall-trophy"><Trophy/></span>
            <span className="hall-champion-what">
                <span id={label} className="menu-label">Season {season.number}</span>
                {championsOf(season).map((champion) => <span key={champion.countryCode} className="hall-champion-name">
                    <CountryFlag code={champion.countryCode}/>
                    {nameOf(champion.countryCode)}
                </span>)}
            </span>
        </div>

        <h3 className="hall-heading">Countries</h3>
        <table className="leaderboard-table hall-table" aria-label={`Countries, season ${season.number}`}>
            <thead>
            <tr>
                <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                <th className="leaderboard-table-head" scope="col">Country</th>
                <th className="leaderboard-table-head leaderboard-table-number" scope="col">Points</th>
            </tr>
            </thead>
            <tbody>
            {listed.map((score) => <ScoreRow key={score.countryCode} score={score} you={score.countryCode === countryCode}/>)}
            {own && <>
                <tr className="standings-gap" aria-hidden="true">
                    <td colSpan={3}/>
                </tr>
                <ScoreRow score={own} you/>
            </>}
            </tbody>
        </table>

        <h3 className="hall-heading">Players</h3>
        <StandingsTable label={`Players, season ${season.number}`}
                        standings={season.standings}
                        accountId={accountId}
                        onOpenPlayer={onOpenPlayer}/>
    </section>
}

function ScoreRow({score, you}: {score: Score, you: boolean}) {
    return <tr className={you ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
               aria-current={you ? "true" : undefined}>
        <td className="leaderboard-entry-index"><RankCoin rank={score.rank} you={you}/></td>
        <td className="leaderboard-entry-country">
            <CountryFlag code={score.countryCode}/>
            {truncate(nameOf(score.countryCode), NAME_MAX_LENGTH)}
        </td>
        <td className="leaderboard-table-number">{score.points}</td>
    </tr>
}

function nameOf(countryCode: string): string {
    return Countries.get(countryCode)?.name ?? countryCode
}
