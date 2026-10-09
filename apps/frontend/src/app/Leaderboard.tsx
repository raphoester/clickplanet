import "./Leaderboard.css"
import {PointerEvent, ReactNode, useEffect, useId, useRef, useState} from "react";
import {Race} from "../backends/standings.ts";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {CountryLine, countryLines, CountryOrder} from "../domain/race.ts";
import {Figure, FIGURES, GUIDE_MS, TAP_HINT_MS} from "./boardFigures.ts";
import {DELTA_HOLD_MS, NO_TILE_DELTAS, signed, TileDelta, TileDeltas} from "../domain/tileDeltas.ts";
import {truncate} from "./truncate.ts";
import Bubble from "./components/Bubble.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import RankCoin from "./components/RankCoin.tsx";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    deltas?: TileDeltas,
    highlight?: Country,
    anthem?: ReactNode,
    race?: Race,
    order?: CountryOrder,
    guided?: boolean,
    onGuided?: () => void,
}

const NAME_MAX_LENGTH = 18

const ORDERS: {order: CountryOrder, label: string}[] = [
    {order: "season", label: "Season"},
    {order: "territory", label: "Territory"},
]

type Shown = {figure: Figure, at: "head" | "leader", ms?: number}

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS
    const race = props.race
    const order = race ? props.order ?? "season" : "territory"
    const [leader, ...rest] = countryLines(props.data, race, order)
    const [shown, setShown] = useState<Shown>()
    const [guiding, setGuiding] = useState(false)

    if (race && rest.length > 0 && props.guided === false && !guiding) {
        setGuiding(true)
        setShown({figure: "points", at: "head", ms: GUIDE_MS})
    }

    const onGuided = props.onGuided
    useEffect(() => {
        if (guiding) onGuided?.()
    }, [guiding, onGuided])

    useEffect(() => {
        if (shown?.ms === undefined) return
        const timer = setTimeout(() => setShown(undefined), shown.ms)
        return () => clearTimeout(timer)
    }, [shown])

    const hint = (figure: Figure, at: Shown["at"]) => ({
        figure,
        open: shown?.figure === figure && shown.at === at,
        onShow: (timed: boolean) => setShown({figure, at, ms: timed ? TAP_HINT_MS : undefined}),
        onHide: () => setShown((now) => now?.figure === figure && now.at === at && now.ms === undefined ? undefined : now),
        onLost: () => setShown((now) => now?.figure === figure && now.at === at ? undefined : now),
    })

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="sr-only" id={titleId}>Leaderboard</h2>

        {leader && <LeaderFrame line={leader}
                                scored={race !== undefined}
                                tilesCount={props.tilesCount}
                                delta={deltas.get(leader.country.code)}
                                isPlayer={leader.country.code === props.highlight?.code}
                                anthem={props.anthem}
                                hint={(figure) => hint(figure, "leader")}/>}

        {rest.length > 0 && <div className="leaderboard-table-container">
            <table className="leaderboard-table">
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">
                        <Hint {...hint("tiles", "head")}>Tiles</Hint>
                    </th>
                    <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-share" scope="col">
                        <Hint {...hint("share", "head")}>% of map</Hint>
                    </th>
                    {race && <>
                        <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-points" scope="col">
                            <Hint {...hint("points", "head")}>Points</Hint>
                        </th>
                        <th className="leaderboard-table-head leaderboard-table-today" scope="col">
                            <Hint {...hint("today", "head")}>Today</Hint>
                        </th>
                    </>}
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
                        <td className="leaderboard-table-number leaderboard-table-share">
                            {share(line.tiles, props.tilesCount)}
                        </td>
                        {race && <>
                            <td className="leaderboard-table-number leaderboard-table-points">{line.points}</td>
                            <td className="leaderboard-table-today"><Today points={line.today}/></td>
                        </>}
                    </tr>
                })}
                </tbody>
            </table>
        </div>}
    </section>
}

export function OrderSwitch({order, onOrder}: {order: CountryOrder, onOrder: (order: CountryOrder) => void}) {
    return <div className="leaderboard-order" role="group" aria-label="Order">
        {ORDERS.map(({order: each, label}) => <button key={each}
                                                      type="button"
                                                      className="leaderboard-order-choice"
                                                      aria-pressed={order === each}
                                                      onClick={() => onOrder(each)}>
            {label}
        </button>)}
    </div>
}

type LeaderFrameProps = {
    line: CountryLine
    scored: boolean
    tilesCount: number
    delta?: TileDelta
    isPlayer: boolean
    anthem?: ReactNode
    hint: (figure: Figure) => HintProps
}

function LeaderFrame({line, scored, tilesCount, delta, isPlayer, anthem, hint}: LeaderFrameProps) {
    const className = isPlayer ? "leader-frame panel-box leader-frame--you" : "leader-frame panel-box"

    return <section className={className}
                    aria-label={`First: ${line.country.name}`}
                    aria-current={isPlayer ? "true" : undefined}>
        <div className="leader-frame-row">
            <span className={isPlayer ? "coin coin-you leader-frame-coin" : "coin coin-1 leader-frame-coin"}>1</span>
            <span className="leader-frame-flag"><CountryFlag code={line.country.code}/></span>
            <span className="leader-frame-who">
                <span className="leader-frame-name">{truncate(line.country.name, NAME_MAX_LENGTH)}</span>
                <span className="leader-frame-line">
                    {scored && <span className="leader-frame-map">
                        <Hint {...hint("share")}>{share(line.tiles, tilesCount)}% of map</Hint>
                    </span>}
                    <span className={tilesClass(delta?.net, "leader-frame-tiles")}>
                        {line.tiles} tiles
                        <DeltaBadge delta={delta}/>
                    </span>
                </span>
            </span>
            {scored
                ? <Hint {...hint("points")} className="leader-frame-share">
                    <span>{line.points}<Today points={line.today}/></span>
                    <span className="leader-frame-share-unit">points</span>
                </Hint>
                : <Hint {...hint("share")} className="leader-frame-share">
                    {share(line.tiles, tilesCount)}
                    <span className="leader-frame-share-unit">% of map</span>
                </Hint>}
        </div>
        {anthem}
    </section>
}

type HintProps = {
    figure: Figure
    open: boolean
    onShow: (timed: boolean) => void
    onHide: () => void
    onLost: () => void
}

function Hint({figure, open, onShow, onHide, onLost, className, children}: HintProps & {className?: string, children: ReactNode}) {
    const anchor = useRef<HTMLButtonElement>(null)
    const pointer = useRef("")
    const described = useId()
    const mouse = (event: PointerEvent) => event.pointerType === "mouse"

    return <>
        <button type="button"
                ref={anchor}
                className={className ? `leaderboard-hint ${className}` : "leaderboard-hint"}
                aria-describedby={described}
                onPointerEnter={(event) => mouse(event) && onShow(false)}
                onPointerLeave={(event) => mouse(event) && onHide()}
                onPointerDown={(event) => pointer.current = event.pointerType}
                onClick={() => {
                    if (pointer.current !== "mouse") onShow(true)
                    pointer.current = ""
                }}>
            {children}
            <span id={described} hidden>{FIGURES[figure]}</span>
        </button>
        {open && <Bubble anchor={anchor} onLost={onLost}>{FIGURES[figure]}</Bubble>}
    </>
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
