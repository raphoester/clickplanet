import "./Leaderboard.css"
import {useId} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {DELTA_HOLD_MS, NO_TILE_DELTAS, signed, TileDeltas} from "../domain/tileDeltas.ts";
import {truncate} from "./truncate.ts";
import CountryFlag from "./components/CountryFlag.tsx";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    deltas?: TileDeltas,
    highlight?: Country,
}

const NAME_MAX_LENGTH = 18

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="menu-section-title" id={titleId}>Leaderboard</h2>

        <div className="leaderboard-table-container">
            <table className="leaderboard-table">
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">Tiles</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">Share</th>
                </tr>
                </thead>

                <tbody>
                {props.data.map((entry, index) => {
                    const isPlayer = entry.country.code === props.highlight?.code
                    const delta = deltas.get(entry.country.code)
                    return <tr key={entry.country.code}
                               className={isPlayer ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
                               aria-current={isPlayer ? "true" : undefined}>
                        <td className="leaderboard-entry-index">{index + 1}</td>
                        <td className="leaderboard-entry-country">
                            <CountryFlag code={entry.country.code}/>
                            {truncate(entry.country.name, NAME_MAX_LENGTH)}
                        </td>
                        <td className={tilesClass(delta?.net)}>
                            {entry.tiles}
                            {delta && <span key={delta.beat}
                                            className="leaderboard-delta"
                                            style={{animationDuration: `${DELTA_HOLD_MS}ms`}}
                                            aria-hidden="true">{signed(delta.net)}</span>}
                        </td>
                        <td className="leaderboard-table-number">
                            {(entry.tiles / props.tilesCount * 100).toFixed(2)}
                        </td>
                    </tr>
                })}
                </tbody>
            </table>
        </div>
    </section>
}

// The count itself takes the badge's colour while it is up, so a row that moved
// reads as one thing rather than a number and a sticker beside it.
function tilesClass(net: number | undefined): string {
    const base = "leaderboard-table-number leaderboard-table-tiles"
    if (net === undefined) return base
    return `${base} ${net > 0 ? "leaderboard-tiles-up" : "leaderboard-tiles-down"}`
}
