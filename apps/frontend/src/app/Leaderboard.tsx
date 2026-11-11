import "./Leaderboard.css"
import {PointerEvent, ReactNode, useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {CountryLine, countryLines, CountryOrder, Ranks, scoringNow, shownOrder} from "../domain/race.ts";
import {ordinalOf} from "../domain/ordinal.ts";
import {sharePercent} from "../domain/territory.ts";
import {Figure, FIGURES, GUIDE_MS, TAP_HINT_MS} from "./boardFigures.ts";
import {DELTA_HOLD_MS, NO_TILE_DELTAS, TileDelta, TileDeltas} from "../domain/tileDeltas.ts";
import {truncate} from "./truncate.ts";
import Bubble from "./components/Bubble.tsx";
import ScrollArea from "./components/ScrollArea.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import {ChevronIcon, SwordsIcon, TrendIcon} from "./components/icons.tsx";
import RankCoin from "./components/RankCoin.tsx";
import {useBoardGuide} from "./boardGuideStore.ts";
import {useRace} from "./standings/raceStore.ts";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    deltas?: TileDeltas,
    highlight?: Country,
    anthem?: ReactNode,
    order?: CountryOrder,
    onOrder?: (order: CountryOrder) => void,
}

const NAME_MAX_LENGTH = 18

const count = new Intl.NumberFormat()
const gained = new Intl.NumberFormat(undefined, {signDisplay: "exceptZero"})

type Shown = {figure: Figure, at: "head" | "leader", ms?: number}

export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()
    const deltas = props.deltas ?? NO_TILE_DELTAS
    const race = useRace((standings) => standings.race)
    const battle = race?.battle
    const gaining = scoringNow(race) !== undefined
    const gainFigure: Figure = battle ? "gainedBattle" : "gained"
    const order = shownOrder(race, props.order)
    // A day places the sum of each country's two ranks: ordered by it, the board shows them.
    const explained = order === "today" && !battle
    const [leader, ...rest] = countryLines(props.data, race, order)
    const [shown, setShown] = useState<Shown>()
    const [guiding, setGuiding] = useState(false)
    const {guided, markGuided} = useBoardGuide()

    if (race && rest.length > 0 && !guided && !guiding) {
        setGuiding(true)
        setShown({figure: "points", at: "head", ms: GUIDE_MS})
    }

    useEffect(() => {
        if (guiding) markGuided()
    }, [guiding, markGuided])

    useEffect(() => {
        if (shown?.ms === undefined) return
        const timer = setTimeout(() => setShown(undefined), shown.ms)
        return () => clearTimeout(timer)
    }, [shown])

    // A bubble open, a touch anywhere puts it away, its own head included. A mouse's bubble follows the mouse.
    const open = shown !== undefined
    useEffect(() => {
        if (!open) return
        const away = (event: globalThis.PointerEvent) => {
            if (event.pointerType !== "mouse") setShown(undefined)
        }
        document.addEventListener("pointerdown", away, true)
        return () => document.removeEventListener("pointerdown", away, true)
    }, [open])

    const hint = (figure: Figure, at: Shown["at"]) => ({
        figure,
        open: shown?.figure === figure && shown.at === at,
        onShow: (timed: boolean) => setShown({figure, at, ms: timed ? TAP_HINT_MS : undefined}),
        onHide: () => setShown((now) => now?.figure === figure && now.at === at && now.ms === undefined ? undefined : now),
        onLost: () => setShown((now) => now?.figure === figure && now.at === at ? undefined : now),
    })

    const onOrder = race ? props.onOrder : undefined
    const sort = (by: CountryOrder): Sort | undefined => onOrder && {on: order === by, pick: () => onOrder(by)}
    const sorted = (by: CountryOrder) => onOrder && order === by ? "descending" : undefined
    const sortToday = gaining ? sort("today") : undefined

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="sr-only" id={titleId}>Leaderboard</h2>

        {leader && <LeaderFrame line={leader}
                                ranks={explained ? leader.ranks : undefined}
                                scored={race !== undefined}
                                battle={battle !== undefined}
                                gainFigure={gainFigure}
                                tilesCount={props.tilesCount}
                                delta={deltas.get(leader.country.code)}
                                isPlayer={leader.country.code === props.highlight?.code}
                                anthem={props.anthem}
                                hint={(figure) => hint(figure, "leader")}/>}

        {rest.length > 0 && <ScrollArea className="leaderboard-table-container">
            <table className="leaderboard-table">
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-tiles" scope="col"
                        aria-sort={sorted("tiles")}>
                        <Hint {...hint("tiles", "head")} sort={sort("tiles")}>Tiles</Hint>
                    </th>
                    {gaining && <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-gain" scope="col"
                                    aria-sort={sorted("gain")}>
                        <Hint {...hint(gainFigure, "head")} sort={sort("gain")}>Gained</Hint>
                    </th>}
                    {race && <>
                        <th className="leaderboard-table-head leaderboard-table-number leaderboard-table-points" scope="col"
                            aria-sort={sorted("points")}>
                            <Hint {...hint("points", "head")} sort={sort("points")}>Points</Hint>
                        </th>
                        <th className="leaderboard-table-head leaderboard-table-today" scope="col"
                            aria-sort={sorted("today")}>
                            {battle
                                ? <Hint {...hint("battle", "head")} sort={sortToday} className="leaderboard-battle">
                                    <SwordsIcon size={12}/><span className="sr-only">Battle</span> ×{battle.times}
                                </Hint>
                                : <Hint {...hint("today", "head")} sort={sortToday}>Today</Hint>}
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
                            <TileCount tiles={line.tiles} delta={delta}/> <MapShare tiles={line.tiles} tilesCount={props.tilesCount}/>
                            {explained && line.ranks && <RankNote rank={line.ranks.tiles} on="tiles"/>}
                        </td>
                        {line.gain !== undefined && <td className="leaderboard-table-number leaderboard-table-gain">
                            <span className={gainClass(line.gain)}>{gained.format(line.gain)}</span>
                            {explained && line.ranks && <RankNote rank={line.ranks.gain} on="gained"/>}
                        </td>}
                        {race && <>
                            <td className="leaderboard-table-number leaderboard-table-points">{count.format(line.points)}</td>
                            <td className="leaderboard-table-today"><Today points={line.today} battle={battle !== undefined}/></td>
                        </>}
                    </tr>
                })}
                </tbody>
            </table>
        </ScrollArea>}
    </section>
}

type LeaderFrameProps = {
    line: CountryLine
    ranks?: Ranks
    scored: boolean
    battle: boolean
    gainFigure: Figure
    tilesCount: number
    delta?: TileDelta
    isPlayer: boolean
    anthem?: ReactNode
    hint: (figure: Figure) => HintProps
}

function LeaderFrame({line, ranks, scored, battle, gainFigure, tilesCount, delta, isPlayer, anthem, hint}: LeaderFrameProps) {
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
                    {line.gain !== undefined && <span className="leader-frame-map">
                        <Hint {...hint(gainFigure)}>
                            <span className={gainClass(line.gain)}>{gained.format(line.gain)}</span> {battle ? "this battle" : "today"}
                        </Hint>
                    </span>}
                    <span className={tilesClass(delta?.net, "leader-frame-tiles")}>
                        <TileCount tiles={line.tiles} delta={delta}/> tiles
                        {scored && <> <MapShare tiles={line.tiles} tilesCount={tilesCount}/></>}
                    </span>
                </span>
                {ranks && <span className="leader-frame-ranks">
                    {ordinalOf(ranks.tiles)} on tiles + {ordinalOf(ranks.gain)} on gained
                </span>}
            </span>
            {scored
                ? <Hint {...hint("points")} className="leader-frame-share">
                    <span>{count.format(line.points)}<Today points={line.today} battle={battle}/></span>
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

type Sort = {on: boolean, pick: () => void}

function Hint({figure, open, onShow, onHide, onLost, sort, className, children}: HintProps & {sort?: Sort, className?: string, children: ReactNode}) {
    const anchor = useRef<HTMLButtonElement>(null)
    const pointer = useRef("")
    // Open as the press began: the press put it away, so its click must not bring it back.
    const wasOpen = useRef(false)
    const described = useId()
    const mouse = (event: PointerEvent) => event.pointerType === "mouse"

    return <>
        <button type="button"
                ref={anchor}
                className={["leaderboard-hint", sort && "leaderboard-sortable", className].filter(Boolean).join(" ")}
                aria-describedby={described}
                onPointerEnter={(event) => mouse(event) && onShow(false)}
                onPointerLeave={(event) => mouse(event) && onHide()}
                onPointerDown={(event) => {
                    pointer.current = event.pointerType
                    wasOpen.current = open && event.pointerType !== "mouse"
                }}
                onClick={() => {
                    sort?.pick()
                    if (pointer.current !== "mouse" && !wasOpen.current) onShow(true)
                    pointer.current = ""
                    wasOpen.current = false
                }}>
            {sort && <span className={sort.on ? "leaderboard-sort leaderboard-sort-on" : "leaderboard-sort"} aria-hidden="true">
                <ChevronIcon size={9}/>
            </span>}
            {children}
            <span id={described} hidden>{FIGURES[figure]}</span>
        </button>
        {open && <Bubble anchor={anchor} onLost={onLost}>{FIGURES[figure]}</Bubble>}
    </>
}

function Today({points, battle}: {points: number, battle: boolean}) {
    if (points <= 0) return null
    return <span className="leaderboard-today">+{points}<span className="sr-only">{battle ? " this battle" : " today"}</span></span>
}

function RankNote({rank, on}: {rank: number, on: string}) {
    return <span className="leaderboard-rank-note">
        {ordinalOf(rank)}<span className="sr-only"> on {on}</span>
    </span>
}

function TileCount({tiles, delta}: {tiles: number, delta?: TileDelta}) {
    return <span className="leaderboard-count">
        {delta && <span key={delta.beat}
                        className="leaderboard-delta"
                        style={{animationDuration: `${DELTA_HOLD_MS}ms`}}
                        aria-hidden="true"><TrendIcon down={delta.net < 0}/></span>}
        {count.format(tiles)}
    </span>
}

function MapShare({tiles, tilesCount}: {tiles: number, tilesCount: number}) {
    return <span className="leaderboard-map-share">{sharePercent(tiles, tilesCount)}%</span>
}

function share(tiles: number, tilesCount: number): string {
    const percent = tiles / tilesCount * 100
    const rounded = percent.toFixed(2)
    return percent > 0 && rounded === "0.00" ? "<0.01" : rounded
}

function gainClass(gain: number): string | undefined {
    if (gain === 0) return undefined
    return gain > 0 ? "leaderboard-gain-up" : "leaderboard-gain-down"
}

function tilesClass(net: number | undefined, base = "leaderboard-table-number leaderboard-table-tiles"): string {
    if (net === undefined) return base
    return `${base} ${net > 0 ? "leaderboard-tiles-up" : "leaderboard-tiles-down"}`
}
