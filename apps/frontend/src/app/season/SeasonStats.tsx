import {CSSProperties, ReactNode, useEffect, useRef, useState} from "react"
import {Season, SeasonTotals} from "../../backends/season.ts"
import {BookStep, Round, Score, Standing, StandingsBackend} from "../../backends/standings.ts"
import {TurnBackend} from "../../backends/turn.ts"
import {Countries} from "../../domain/countries.ts"
import {spanLine} from "../../domain/seasonClock.ts"
import {
    Barred,
    BattleFlagRow,
    battleFlagRowsOf,
    BigNumber,
    bigNumbersOf,
    playerRowsOf,
    seasonFlagRowsOf,
    Slide,
    slidesOf,
} from "../../domain/seasonStats.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import RankCoin from "../components/RankCoin.tsx"
import {BackIcon} from "../components/icons.tsx"
import TitleBadge from "../titles/TitleBadge.tsx"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import RaceChart from "./RaceChart.tsx"
import "./SeasonStats.css"

export type SeasonStatsProps = {
    season: Season
    standings: Pick<StandingsBackend, "scores" | "board" | "round" | "bookHistory">
    turn: Pick<TurnBackend, "seasonTotals">
    onBack: () => void
}

const numbers = new Intl.NumberFormat("en-US")

export default function SeasonStats({season, standings, turn, onBack}: SeasonStatsProps) {
    const table = useAnswer(`table:${season.bookId}`, () => standings.scores(season.bookId))
    const board = useAnswer(`board:${season.bookId}`, () => standings.board(season.bookId))
    const finale = useAnswer(`finale:${season.finaleRoundId}`, async () => season.finaleRoundId ? standings.round(season.finaleRoundId) : undefined)
    const totals = useAnswer(`totals:${season.number}`, () => turn.seasonTotals(season.number))
    const history = useAnswer(`history:${season.bookId}`, () => standings.bookHistory(season.bookId))
    const slides = slidesOf(season)
    const track = useRef<HTMLOListElement>(null)
    const [shown, setShown] = useState(0)

    const count = slides.length
    const go = (index: number) => scrollToSlide(track.current, Math.max(0, Math.min(count - 1, index)))

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === "ArrowRight") scrollToSlide(track.current, Math.min(count - 1, shown + 1))
            if (event.key === "ArrowLeft") scrollToSlide(track.current, Math.max(0, shown - 1))
        }
        document.addEventListener("keydown", onKeyDown)
        return () => document.removeEventListener("keydown", onKeyDown)
    }, [count, shown])

    return <div className="season-stats">
        <div className="season-stats-head">
            <button type="button" className="button button-mini button-secondary season-stats-back" onClick={onBack}>
                <BackIcon size={16}/>
                <span>Hall of Fame</span>
            </button>
            <span className="season-stats-dates">{spanLine(season)}</span>
        </div>

        <ol ref={track}
            className="season-stats-track"
            aria-label={`Season ${season.number} in numbers`}
            onScroll={(e) => setShown(Math.round(e.currentTarget.scrollLeft / Math.max(1, e.currentTarget.clientWidth)))}>
            {slides.map((slide, i) => <li key={slide}
                                          className="season-stats-slide"
                                          aria-roledescription="slide"
                                          aria-label={`${i + 1} of ${slides.length}`}
                                          aria-hidden={i !== shown}>
                <SlideOf slide={slide} season={season} table={table} board={board} finale={finale} totals={totals} history={history}
                         active={i === shown}/>
            </li>)}
        </ol>

        <div className="season-stats-nav">
            <button type="button" className="icon-button" aria-label="Previous" disabled={shown === 0} onClick={() => go(shown - 1)}>
                <BackIcon size={20}/>
            </button>
            <span className="season-stats-dots">
                {slides.map((slide, i) => <button key={slide}
                                                  type="button"
                                                  className={i === shown ? "season-stats-dot season-stats-dot--on" : "season-stats-dot"}
                                                  aria-label={`Slide ${i + 1} of ${slides.length}`}
                                                  aria-current={i === shown ? "true" : undefined}
                                                  onClick={() => go(i)}/>)}
            </span>
            <button type="button" className="icon-button season-stats-next" aria-label="Next" disabled={shown === slides.length - 1} onClick={() => go(shown + 1)}>
                <BackIcon size={20}/>
            </button>
        </div>
    </div>
}

type SlideProps = {
    slide: Slide
    season: Season
    table?: Score[]
    board?: Standing[]
    finale?: Round
    totals?: SeasonTotals
    history?: BookStep[]
    active: boolean
}

function SlideOf({slide, season, table, board, finale, totals, history, active}: SlideProps) {
    switch (slide) {
        case "race":
            return <Frame kicker="Season" head="The race">
                {history && <RaceChart steps={history} active={active}/>}
            </Frame>
        case "flags":
            return <Frame kicker="Season" head="Top flags" unit="Points">
                {table && <FlagRows rows={seasonFlagRowsOf(table)}
                                    value={(row) => row.points}
                                    aside={(row) => row.roundsWon > 0 ? `${row.roundsWon} ${row.roundsWon === 1 ? "day" : "days"} won` : ""}/>}
            </Frame>
        case "players":
            return <Frame kicker="Season" head="Top players" unit="Tiles">
                {board && <PlayerRows rows={playerRowsOf(board)}/>}
            </Frame>
        case "battleFlags":
            return <Frame kicker="Final Battle" battle head="Top flags" unit={finale && finale.times > 1 ? `Points ×${finale.times}` : "Points"}>
                {finale && <FlagRows<BattleFlagRow> rows={battleFlagRowsOf(finale)}
                                                    value={(row) => row.points}
                                                    aside={(row) => `${numbers.format(row.tiles)} tiles`}/>}
            </Frame>
        case "battlePlayers":
            return <Frame kicker="Final Battle" battle head="Top players" unit="Tiles">
                {finale && <PlayerRows rows={playerRowsOf(finale.players)}/>}
            </Frame>
        case "numbers":
            return <Frame kicker="Season" head="In numbers">
                <Numbers numbers={bigNumbersOf(season, totals, table, finale)}/>
            </Frame>
    }
}

function Frame({kicker, battle, head, unit, children}: {kicker: string, battle?: boolean, head: string, unit?: string, children: ReactNode}) {
    return <>
        <div className="season-stats-title">
            <span className={battle ? "season-stats-kicker season-stats-kicker--battle" : "season-stats-kicker"}>{kicker}</span>
            <h3 className="season-stats-head-line">{head}</h3>
            {unit && <span className="season-stats-unit">{unit}</span>}
        </div>
        {children}
    </>
}

function FlagRows<T extends Barred<{rank: number, countryCode: string}>>({rows, value, aside}: {
    rows: T[],
    value: (row: T) => number,
    aside: (row: T) => string,
}) {
    if (rows.length === 0) return <p className="season-stats-none">Nobody scored</p>

    return <ol className="season-stats-rows">
        {rows.map((row, i) => <li key={row.countryCode} className="season-stats-row" style={{"--share": row.width, "--enter": i} as CSSProperties}>
            <RankCoin rank={row.rank} you={false}/>
            <span className="season-stats-who">
                <CountryFlag code={row.countryCode}/>
                <span className="season-stats-name">{Countries.get(row.countryCode)?.name ?? row.countryCode}</span>
                <span className="season-stats-aside">{aside(row)}</span>
            </span>
            <span className="season-stats-value">{numbers.format(value(row))}</span>
        </li>)}
    </ol>
}

function PlayerRows({rows}: {rows: Barred<Standing>[]}) {
    const words = useWordFilter()
    if (rows.length === 0) return <p className="season-stats-none">Nobody played</p>

    return <ol className="season-stats-rows">
        {rows.map((row, i) => <li key={row.accountId} className="season-stats-row" style={{"--share": row.width, "--enter": i} as CSSProperties}>
            <RankCoin rank={row.rank} you={false}/>
            <span className="season-stats-who">
                <span role="img" aria-label={Countries.get(row.countryCode)?.name ?? row.countryCode}>
                    <CountryFlag code={row.countryCode}/>
                </span>
                <span className="season-stats-name season-stats-player"
                      style={authorStyle({color: row.color, guest: false})}
                      title={words.masked(row.name)}>{words.masked(row.name)}</span>
                <TitleBadge title={row.wornTitle} size={18}/>
            </span>
            <span className="season-stats-value">{numbers.format(row.tiles)}</span>
        </li>)}
    </ol>
}

function Numbers({numbers: shown}: {numbers: BigNumber[]}) {
    return <dl className="season-stats-numbers">
        {shown.map((number, i) => <div key={number.label}
                                       className={i === 0 ? "season-stats-number season-stats-number--hero" : "season-stats-number"}
                                       style={{"--enter": i} as CSSProperties}>
            <dt>{number.label}</dt>
            <dd>{numbers.format(number.value)}</dd>
        </div>)}
    </dl>
}

function scrollToSlide(track: HTMLOListElement | null, index: number) {
    track?.scrollTo({left: index * track.clientWidth, behavior: "smooth"})
}

// One read per key, kept while the key stays; none while it runs, or when it failed.
function useAnswer<T>(key: string, read: () => Promise<T>): T | undefined {
    const latest = useRef(read)
    latest.current = read
    const [answer, setAnswer] = useState<{key: string, value: T}>()

    useEffect(() => {
        let stale = false
        latest.current().then(
            (value) => {
                if (!stale) setAnswer({key, value})
            },
            (e) => console.error(`Could not read ${key}`, e),
        )
        return () => {
            stale = true
        }
    }, [key])

    return answer?.key === key ? answer.value : undefined
}
