import "./Leaderboard.css"
import {ReactNode, useId} from "react";
import {Country} from "../domain/countries.ts";
import {factor} from "../domain/clickPrice.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {DELTA_HOLD_MS, NO_TILE_DELTAS, signed, TileDelta, TileDeltas} from "../domain/tileDeltas.ts";
import {slowdownAt, TollStep} from "../domain/toll.ts";
import {truncate} from "./truncate.ts";
import CountryFlag from "./components/CountryFlag.tsx";
import {HourglassIcon} from "./components/icons.tsx";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    deltas?: TileDeltas,
    highlight?: Country,
    toll?: readonly TollStep[],
    anthem?: ReactNode,
}

const NAME_MAX_LENGTH = 18

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS
    const [leader, ...rest] = props.data

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="sr-only" id={titleId}>Leaderboard</h2>

        {leader && <LeaderFrame entry={leader}
                                tilesCount={props.tilesCount}
                                delta={deltas.get(leader.country.code)}
                                isPlayer={leader.country.code === props.highlight?.code}
                                toll={props.toll ?? []}
                                anthem={props.anthem}/>}

        {rest.length > 0 && <div className="leaderboard-table-container">
            <table className="leaderboard-table">
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
                {rest.map((entry, index) => {
                    const isPlayer = entry.country.code === props.highlight?.code
                    const delta = deltas.get(entry.country.code)
                    return <tr key={entry.country.code}
                               className={isPlayer ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
                               aria-current={isPlayer ? "true" : undefined}>
                        <td className="leaderboard-entry-index">{rankBadge(index + 2, isPlayer)}</td>
                        <td className="leaderboard-entry-country">
                            <CountryFlag code={entry.country.code}/>
                            {truncate(entry.country.name, NAME_MAX_LENGTH)}
                        </td>
                        <td className={tilesClass(delta?.net)}>
                            {entry.tiles}
                            <DeltaBadge delta={delta}/>
                        </td>
                        <td className="leaderboard-table-number leaderboard-table-share">
                            {share(entry.tiles, props.tilesCount)}
                        </td>
                    </tr>
                })}
                </tbody>
            </table>
        </div>}
    </section>
}

type LeaderFrameProps = {
    entry: LeaderboardEntry
    tilesCount: number
    delta?: TileDelta
    isPlayer: boolean
    toll: readonly TollStep[]
    anthem?: ReactNode
}

function LeaderFrame({entry, tilesCount, delta, isPlayer, toll, anthem}: LeaderFrameProps) {
    const slowdown = slowdownAt(toll, entry.tiles / tilesCount)
    const className = isPlayer ? "leader-frame panel-box leader-frame--you" : "leader-frame panel-box"

    return <section className={className}
                    aria-label={`First: ${entry.country.name}`}
                    aria-current={isPlayer ? "true" : undefined}>
        <div className="leader-frame-row">
            <span className={isPlayer ? "coin coin-you leader-frame-coin" : "coin coin-1 leader-frame-coin"}>1</span>
            <span className="leader-frame-flag"><CountryFlag code={entry.country.code}/></span>
            <span className="leader-frame-who">
                <span className="leader-frame-name">{truncate(entry.country.name, NAME_MAX_LENGTH)}</span>
                <span className={tilesClass(delta?.net, "leader-frame-tiles")}>
                    {entry.tiles} tiles
                    <DeltaBadge delta={delta}/>
                </span>
            </span>
            <span className="leader-frame-share">
                {share(entry.tiles, tilesCount)}
                <span className="leader-frame-share-unit">% of map</span>
            </span>
        </div>
        {slowdown > 1 && <span className="leader-frame-toll">
            <HourglassIcon/>
            Refills {factor(slowdown)}× slower
        </span>}
        {anthem}
    </section>
}

function DeltaBadge({delta}: {delta?: TileDelta}) {
    if (!delta) return null
    return <span key={delta.beat}
                 className="leaderboard-delta"
                 style={{animationDuration: `${DELTA_HOLD_MS}ms`}}
                 aria-hidden="true">{signed(delta.net)}</span>
}

function rankBadge(rank: number, isPlayer: boolean) {
    if (isPlayer) return <span className="coin coin-you">{rank}</span>
    if (rank <= 3) return <span className={`coin coin-${rank}`}>{rank}</span>
    return <span className="leaderboard-rank">{rank}</span>
}

function share(tiles: number, tilesCount: number): string {
    const percent = tiles / tilesCount * 100
    const rounded = percent.toFixed(2)
    return percent > 0 && rounded === "0.00" ? "<0.01" : rounded
}

function tilesClass(net: number | undefined, base = "leaderboard-table-number leaderboard-table-tiles"): string {
    if (net === undefined) return base
    return `${base} ${net > 0 ? "leaderboard-tiles-up" : "leaderboard-tiles-down"}`
}
