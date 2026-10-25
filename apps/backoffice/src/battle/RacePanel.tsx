import {Country} from "../components/Country.tsx"
import {clockOf} from "../components/time.ts"
import {Race} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {wholeOf} from "../home/activity.ts"

const SHOWN = 10
const PERCENT = new Intl.NumberFormat("en", {style: "percent", maximumFractionDigits: 1})

// The round the countries race for now, and the season's points, as the players see them.
export function RacePanel({race, now}: {race: Race, now: Date}) {
    const round = race.round
    const ends = round === undefined ? undefined : new Date(Number(round.endsAtUnixMs))
    const closed = race.closed
    const leader = closed?.standings[0]

    return (
        <section className="panel" aria-label="Race">
            <div className="panel-head">
                <h2>{round === undefined ? "Race" : `Round ${round.number}${round.finale ? ", the finale" : ""}`}</h2>
                {ends !== undefined && <span className="muted">ends {clockOf(ends)}, {leftOf(ends, now)}</span>}
            </div>
            {closed !== undefined && leader !== undefined && (
                <p className="muted">
                    Round {closed.number} closed at {clockOf(new Date(Number(closed.endedAtUnixMs)))}:{" "}
                    <Country id={leader.countryId}/> won
                </p>
            )}
            {round !== undefined && round.standings.length > 0 && (
                <table className="table">
                    <thead>
                        <tr>
                            <th scope="col" className="num">#</th>
                            <th scope="col">Country</th>
                            <th scope="col" className="num">Share</th>
                            <th scope="col" className="num">Points</th>
                        </tr>
                    </thead>
                    <tbody>
                        {round.standings.slice(0, SHOWN).map(standing => (
                            <tr key={standing.countryId}>
                                <td className="num">{standing.rank}</td>
                                <td><Country id={standing.countryId}/></td>
                                <td className="num">{PERCENT.format(standing.share)}</td>
                                <td className="num">{wholeOf(standing.points)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
            {race.scores.length > 0 && (
                <table className="table">
                    <caption className="table-caption">The season</caption>
                    <thead>
                        <tr>
                            <th scope="col" className="num">#</th>
                            <th scope="col">Country</th>
                            <th scope="col" className="num">Points</th>
                            <th scope="col" className="num">Rounds won</th>
                        </tr>
                    </thead>
                    <tbody>
                        {race.scores.slice(0, SHOWN).map(score => (
                            <tr key={score.countryId}>
                                <td className="num">{score.rank}</td>
                                <td><Country id={score.countryId}/></td>
                                <td className="num">{wholeOf(score.points)}</td>
                                <td className="num">{wholeOf(score.roundsWon)}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </section>
    )
}

function leftOf(ends: Date, now: Date): string {
    const minutes = Math.round((ends.getTime() - now.getTime()) / 60_000)
    if (minutes <= 0) return "now"
    if (minutes < 60) return `in ${minutes} min`
    return `in ${Math.floor(minutes / 60)} h ${String(minutes % 60).padStart(2, "0")} min`
}
