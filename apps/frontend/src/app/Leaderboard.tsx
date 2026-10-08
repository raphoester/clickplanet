import "./Leaderboard.css"
import {ReactNode, useId} from "react";
import {Race} from "../backends/standings.ts";
import {Country} from "../domain/countries.ts";
import {factor} from "../domain/clickPrice.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {CountryLine, countryLines, CountryOrder} from "../domain/race.ts";
import {DELTA_HOLD_MS, NO_TILE_DELTAS, signed, TileDelta, TileDeltas} from "../domain/tileDeltas.ts";
import {slowdownAt, TollStep} from "../domain/toll.ts";
import {truncate} from "./truncate.ts";
import CountryFlag from "./components/CountryFlag.tsx";
import {HourglassIcon} from "./components/icons.tsx";
import RankCoin from "./components/RankCoin.tsx";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    deltas?: TileDeltas,
    highlight?: Country,
    toll?: readonly TollStep[],
    anthem?: ReactNode,
    race?: Race,
    order?: CountryOrder,
    onOrder?: (order: CountryOrder) => void,
}

const NAME_MAX_LENGTH = 18

const ORDERS: {order: CountryOrder, label: string}[] = [
    {order: "season", label: "Season"},
    {order: "territory", label: "Territory"},
]

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS
    const race = props.race
    const order = race ? props.order ?? "season" : "territory"
    const [leader, ...rest] = countryLines(props.data, race, order)

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="sr-only" id={titleId}>Leaderboard</h2>

        {race && props.onOrder && <div className="leaderboard-order" role="group" aria-label="Order">
            {ORDERS.map(({order: each, label}) => <button key={each}
                                                          type="button"
                                                          className={`button button-mini leaderboard-order-choice${order === each ? " button-secondary" : ""}`}
                                                          aria-pressed={order === each}
                                                          onClick={() => props.onOrder?.(each)}>
                {label}
            </button>)}
        </div>}

        {leader && <LeaderFrame line={leader}
                                scored={race !== undefined}
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
                    {race
                        ? <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-points"
                              scope="col">Points</th>
                        : <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-share"
                              scope="col">% of map
                        </th>}
                </tr>
                </thead>

                <tbody>
                {rest.map((line, index) => {
                    const isPlayer = line.country.code === props.highlight?.code
                    const delta = deltas.get(line.country.code)
                    return <tr key={line.country.code}
                               className={isPlayer ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
                               aria-current={isPlayer ? "true" : undefined}>
                        <td className="leaderboard-entry-index"><RankCoin rank={index + 2} you={isPlayer}/></td>
                        <td className="leaderboard-entry-country">
                            <CountryFlag code={line.country.code}/>
                            {truncate(line.country.name, NAME_MAX_LENGTH)}
                        </td>
                        <td className={tilesClass(delta?.net)}>
                            {line.tiles}
                            <DeltaBadge delta={delta}/>
                        </td>
                        {race
                            ? <td className="leaderboard-table-number leaderboard-table-points">
                                {line.points}
                                <Today points={line.today}/>
                            </td>
                            : <td className="leaderboard-table-number leaderboard-table-share">
                                {share(line.tiles, props.tilesCount)}
                            </td>}
                    </tr>
                })}
                </tbody>
            </table>
        </div>}
    </section>
}

type LeaderFrameProps = {
    line: CountryLine
    scored: boolean
    tilesCount: number
    delta?: TileDelta
    isPlayer: boolean
    toll: readonly TollStep[]
    anthem?: ReactNode
}

function LeaderFrame({line, scored, tilesCount, delta, isPlayer, toll, anthem}: LeaderFrameProps) {
    const slowdown = slowdownAt(toll, line.tiles / tilesCount)
    const className = isPlayer ? "leader-frame panel-box leader-frame--you" : "leader-frame panel-box"

    return <section className={className}
                    aria-label={`First: ${line.country.name}`}
                    aria-current={isPlayer ? "true" : undefined}>
        <div className="leader-frame-row">
            <span className={isPlayer ? "coin coin-you leader-frame-coin" : "coin coin-1 leader-frame-coin"}>1</span>
            <span className="leader-frame-flag"><CountryFlag code={line.country.code}/></span>
            <span className="leader-frame-who">
                <span className="leader-frame-name">{truncate(line.country.name, NAME_MAX_LENGTH)}</span>
                <span className={tilesClass(delta?.net, "leader-frame-tiles")}>
                    {line.tiles} tiles
                    <DeltaBadge delta={delta}/>
                </span>
            </span>
            {scored
                ? <span className="leader-frame-share">
                    <span>{line.points}<Today points={line.today}/></span>
                    <span className="leader-frame-share-unit">points</span>
                </span>
                : <span className="leader-frame-share">
                    {share(line.tiles, tilesCount)}
                    <span className="leader-frame-share-unit">% of map</span>
                </span>}
        </div>
        {slowdown > 1 && <span className="leader-frame-toll">
            <HourglassIcon/>
            Refills {factor(slowdown)}× slower
        </span>}
        {anthem}
    </section>
}

function Today({points}: {points: number}) {
    if (points <= 0) return null
    return <span className="leaderboard-today">+{points}<span className="sr-only"> today</span></span>
}

function DeltaBadge({delta}: {delta?: TileDelta}) {
    if (!delta) return null
    return <span key={delta.beat}
                 className="leaderboard-delta"
                 style={{animationDuration: `${DELTA_HOLD_MS}ms`}}
                 aria-hidden="true">{signed(delta.net)}</span>
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
