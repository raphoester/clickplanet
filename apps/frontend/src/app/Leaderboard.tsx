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

/* The copy around it is English, so the grouping is too: "257.948 tiles on the
   map" would read as a decimal to the very readers the sentence is written for. */
const grouped = new Intl.NumberFormat("en-US")

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const totalId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="menu-section-title" id={titleId}>Leaderboard</h2>

        {/* Names the denominator both number columns are counted against, once,
            instead of repeating a unit on every line. */}
        {props.tilesCount > 0 &&
            <p className="leaderboard-total" id={totalId}>
                {grouped.format(props.tilesCount)} tiles on the map
            </p>}

        <div className="leaderboard-table-container">
            <table className="leaderboard-table"
                   aria-describedby={props.tilesCount > 0 ? totalId : undefined}>
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">Tiles</th>
                    <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-share"
                        scope="col">% of map
                    </th>
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
                        <td className="leaderboard-table-number leaderboard-table-share">
                            {share(entry.tiles, props.tilesCount)}
                        </td>
                    </tr>
                })}
                </tbody>
            </table>
        </div>
    </section>
}

/* A country holding a couple of hundred tiles out of a quarter of a million
   rounds to "0.00", which reads as none at all. Say "small" instead of "none". */
function share(tiles: number, tilesCount: number): string {
    const percent = tiles / tilesCount * 100
    const rounded = percent.toFixed(2)
    return percent > 0 && rounded === "0.00" ? "<0.01" : rounded
}

function tilesClass(net: number | undefined): string {
    const base = "leaderboard-table-number leaderboard-table-tiles"
    if (net === undefined) return base
    return `${base} ${net > 0 ? "leaderboard-tiles-up" : "leaderboard-tiles-down"}`
}
