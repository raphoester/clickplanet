import {useEffect, useMemo, useRef, useState} from "react"
import {ReplayLandmasses} from "../../backends/replayBackend.ts"
import {HistoryBackend} from "../../backends/historyBackend.ts"
import {Season} from "../../backends/season.ts"
import {TurnBackend} from "../../backends/turn.ts"
import {LeaderboardEntry} from "../../domain/leaderboard.ts"
import {MapHistory} from "../../domain/mapHistory.ts"
import type {BoardAsset} from "../../domain/board.ts"
import {ErasReel, piecesOf} from "../../domain/eras.ts"
import type {TileCells} from "../../domain/latticeCells.ts"
import {createGlobe} from "../viewer/globe.ts"
import {boardNamed} from "../viewer/boards.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {PauseIcon, PlayIcon} from "../components/icons.tsx"
import {t, translator} from "../../i18n/page.ts"
import {dayAndTime} from "../season/dates.ts"
import {drawHeat, heatOf, lookOfThePage} from "./heatmap.ts"
import "./SeasonMap.css"

// A board's tiles, the cells they cover and their landmasses, as the archive draws them.
export type BoardGeometry = {board: BoardAsset, positions: Float32Array, cells: TileCells, landmasses: ReplayLandmasses}

export type SeasonMapProps = {
    turn: Pick<TurnBackend, "follow" | "seasonReplay">
    // Each board's, by its name: a history can cross a change of board.
    geometry: (board: string) => Promise<BoardGeometry>
}

type Part = "season" | "finale"
type View = "replay" | "heat"

// Frames a second while it plays: a season of 30-minute frames in about 48 s, a finale of 10-second frames in 30 s.
const FPS = 24

// The caption of a change of board shows this many frames from the cut: a second and a half of play.
const CUT_CAPTION_FRAMES = 36

type Geometries = ReadonlyMap<string, BoardGeometry>

// Opens on the latest season that ended; a switch picks another once there are several.
export default function SeasonMap({turn, geometry}: SeasonMapProps) {
    const ended = useEnded(turn)
    const [picked, setPicked] = useState<number>()
    const season = picked ?? ended?.at(-1)?.number
    const [part, setPart] = useState<Part>("season")
    const [view, setView] = useState<View>("replay")
    const [read, setRead] = useState<{key: string, history?: MapHistory}>()
    // Each board of the history, read; none when one could not be: a board this page does not carry, or a failed read.
    const [geometries, setGeometries] = useState<{history: MapHistory, boards?: Geometries}>()
    const key = `${season}/${part}`
    const history = read?.key === key ? read.history : undefined
    const loading = season !== undefined && read?.key !== key

    // Every board the history's eras play, each once.
    useEffect(() => {
        if (history === undefined) return
        let stale = false
        const played = [...new Set(piecesOf(history).map(({board}) => board))]
        const unknown = played.filter((board) => boardNamed(board) === undefined)
        if (unknown.length > 0) {
            console.error(`the season's replay plays ${unknown.join(", ")}, which this page does not carry`)
            setGeometries({history})
            return
        }
        Promise.all(played.map(geometry)).then(
            (read) => {
                if (!stale) setGeometries({history, boards: new Map(read.map((one) => [one.board.name, one]))})
            },
            (e) => {
                console.error("Could not read the boards of the season's replay", e)
                if (!stale) setGeometries({history})
            },
        )
        return () => {
            stale = true
        }
    }, [history, geometry])
    const boards = geometries?.history === history ? geometries?.boards : undefined
    const unreadable = history !== undefined && geometries?.history === history && boards === undefined

    useEffect(() => {
        if (season === undefined) return
        let stale = false
        turn.seasonReplay(season, part === "finale").then(
            (answer) => {
                if (!stale) setRead({key: `${season}/${part}`, history: answer})
            },
            (e) => {
                console.error("Could not read the season's replay", e)
                if (!stale) setRead({key: `${season}/${part}`})
            },
        )
        return () => {
            stale = true
        }
    }, [turn, season, part])

    const reel = useMemo(() => history && boards && history.frames.length > 0
        ? new ErasReel(history, (board) => boards.get(board)!.board.tiles) : undefined, [history, boards])

    return <div className="season-map">
        <header className="panel season-map-head">
            <a className="button button-mini button-secondary" href="/play">{t("archive.backToGame")}</a>
            {ended && ended.length > 1 && season !== undefined
                ? <Switch label={t("archive.seasonPicker")} value={String(season)} onChange={(value) => setPicked(Number(value))}
                          choices={ended.map((kept) => [String(kept.number), t("season.name", {season: kept.number})])}/>
                : <h1 className="season-map-title">{season === undefined ? t("archive.title") : t("season.name", {season})}</h1>}
            <Switch label={t("archive.window")} value={part} onChange={setPart}
                    choices={[["season", t("archive.wholeSeason")], ["finale", t("finale.name")]]}/>
            <Switch label={t("archive.view")} value={view} onChange={setView}
                    choices={[["replay", t("archive.replay")], ["heat", t("archive.heatmap")]]}/>
        </header>

        <div className="season-map-stage">
            {ended?.length === 0 && <p className="season-map-note">{t("archive.noSeasonEnded")}</p>}
            {loading && <p className="season-map-note">{t("archive.rebuilding")}</p>}
            {season !== undefined && !loading && (!history || unreadable) && <p className="season-map-note">{t("archive.noReplay")}</p>}
            {reel && boards && view === "replay" && <Replay key={key} reel={reel} boards={boards}/>}
            {history && boards && view === "heat" && <Heat key={key} history={history} boards={boards}/>}
        </div>
    </div>
}

// The seasons that ended, from the gamemaster's timeline; none known before it answers.
function useEnded(turn: Pick<TurnBackend, "follow">): Season[] | undefined {
    const [ended, setEnded] = useState<Season[]>()

    useEffect(() => turn.follow({
        onTurn: () => {},
        onBattles: () => {},
        onSeasons: (seasons) => setEnded(seasons.ended),
        onDebrief: () => {},
    }), [turn])

    return ended
}

function Switch<T extends string>({label, value, onChange, choices}: {
    label: string,
    value: T,
    onChange: (value: T) => void,
    choices: [T, string][],
}) {
    return <div className="season-map-switch" role="radiogroup" aria-label={label}>
        {choices.map(([choice, name]) => <button key={choice}
                                                 type="button"
                                                 role="radio"
                                                 aria-checked={choice === value}
                                                 className={choice === value ? "season-map-choice season-map-choice--on" : "season-map-choice"}
                                                 onClick={() => onChange(choice)}>{name}</button>)}
    </div>
}

// Each era is drawn on its own board: past a cut, the globe is built again on the next one, the map cleared and drawn
// whole, and a caption says so while the cut is passed.
function Replay({reel, boards}: {reel: ErasReel, boards: Geometries}) {
    const stage = useRef<HTMLDivElement>(null)
    const backend = useRef<HistoryBackend>()
    const [frame, setFrame] = useState(0)
    const [playing, setPlaying] = useState(false)
    const [leaders, setLeaders] = useState<LeaderboardEntry[]>([])
    const last = reel.length - 1
    const latest = useRef(frame)
    latest.current = frame
    const {piece} = reel.pieceAt(frame)
    const cuts = useMemo(() => reel.cuts(), [reel])
    const passing = reel.passingCut(frame, CUT_CAPTION_FRAMES)

    useEffect(() => {
        const container = stage.current
        const geometry = boards.get(piece.board)
        if (!container || !geometry) return
        const aborted = new AbortController()
        const shown = new HistoryBackend(piece.reel, geometry.landmasses)
        shown.seek(latest.current - piece.from)
        backend.current = shown
        const none = () => {}
        const made = createGlobe({
            board: geometry.board,
            tileClicker: {clickTile: () => Promise.resolve()},
            ownershipsGetter: shown,
            updatesListener: shown,
            container,
            country: {code: ""},
            mapView: "split",
            rendering: "sharp",
            onLeaderboardChange: (entries) => setLeaders(entries.slice(0, 5)),
            onLoadProgress: none, onRateLimited: none, onFrozen: none, onVPNBlocked: none, onBanned: none,
            onSessionUnavailable: none, onAllegianceLocked: none, onOutOfReach: none, onBonusWon: none, onCharges: none,
            onRules: none, onSwitchesChange: none, onBombDropped: none, onArmedChange: none,
            signal: aborted.signal,
        })
        made.catch((e) => {
            if (!aborted.signal.aborted) console.error("could not draw the season's map", e)
        })
        return () => {
            aborted.abort()
            backend.current = undefined
            made.then((globe) => globe.dispose(), () => {})
        }
    }, [piece, boards])

    useEffect(() => {
        backend.current?.seek(reel.pieceAt(frame).frame)
    }, [reel, frame])

    useEffect(() => {
        if (!playing) return
        const timer = setInterval(() => setFrame((at) => {
            if (at >= last) {
                setPlaying(false)
                return at
            }
            return at + 1
        }), 1000 / FPS)
        return () => clearInterval(timer)
    }, [playing, last])

    return <>
        <div ref={stage} className="season-map-globe"/>
        {passing && <p className="panel season-map-cut" role="status">{t("archive.newMap")}</p>}
        <ol className="panel season-map-leaders" aria-label={t("archive.mostTiles")}>
            {leaders.map((entry, i) => <li key={entry.country.code} className="season-map-leader">
                <span className="season-map-rank">{i + 1}</span>
                <CountryFlag code={entry.country.code}/>
                <span className="season-map-leader-name">{translator().country(entry.country.code)}</span>
                <span className="season-map-leader-tiles">{translator().number(entry.tiles)}</span>
            </li>)}
        </ol>
        <footer className="panel season-map-controls">
            <button type="button"
                    className="icon-button"
                    aria-label={t(playing ? "archive.pause" : "archive.play")}
                    onClick={() => {
                        if (!playing && frame >= last) setFrame(0)
                        setPlaying(!playing)
                    }}>
                {playing ? <PauseIcon/> : <PlayIcon/>}
            </button>
            <div className="season-map-track">
                <input className="season-map-cursor"
                       type="range"
                       min={0}
                       max={last}
                       value={frame}
                       aria-label={t("archive.time")}
                       onChange={(e) => {
                           setPlaying(false)
                           setFrame(Number(e.target.value))
                       }}/>
                {cuts.map((cut) => <span key={cut.frame}
                                         className="season-map-mark"
                                         style={{"--at": last > 0 ? cut.frame / last : 0} as React.CSSProperties}
                                         title={t("archive.newMapAt", {time: dayAndTime(cut.at)})}/>)}
            </div>
            <span className="season-map-time">{dayAndTime(reel.atOf(frame))}</span>
        </footer>
    </>
}

const HEAT_WIDTH = 4096
const HEAT_HEIGHT = 2048

// Three copies side by side, the middle one shown: the scroll jumps a whole map at either edge, so it goes round
// forever. It opens on longitude 0. A drag or the wheel scrolls it sideways. Each era's hits are laid where they were
// made, on the finest board among them.
function Heat({history, boards}: {history: MapHistory, boards: Geometries}) {
    const scroller = useRef<HTMLDivElement>(null)
    const copies = [useRef<HTMLCanvasElement>(null), useRef<HTMLCanvasElement>(null), useRef<HTMLCanvasElement>(null)]
    const [first, second, third] = copies

    useEffect(() => {
        const drawn = first.current
        if (!drawn) return
        const eras = piecesOf(history).flatMap(({era, board}) => {
            const geometry = boards.get(board)
            if (geometry === undefined) return []
            const {board: {name, spacing}, positions, cells} = geometry
            return [{board: {name, spacing, positions, cells}, hits: history.hits[era] ?? new Map<number, number>()}]
        })
        if (eras.length > 0) drawHeat(drawn, heatOf(eras), lookOfThePage())
        for (const copy of [second.current, third.current]) copy?.getContext("2d")?.drawImage(drawn, 0, 0)
        const list = scroller.current
        if (list) list.scrollLeft = drawn.clientWidth * 1.5 - list.clientWidth / 2
    }, [history, boards, first, second, third])

    useEffect(() => {
        const list = scroller.current
        if (!list) return
        const onWheel = (event: WheelEvent) => {
            if (Math.abs(event.deltaY) <= Math.abs(event.deltaX)) return
            event.preventDefault()
            list.scrollLeft += event.deltaY
        }
        list.addEventListener("wheel", onWheel, {passive: false})
        return () => list.removeEventListener("wheel", onWheel)
    }, [])

    const wrap = () => {
        const list = scroller.current, map = first.current?.clientWidth ?? 0
        if (!list || map === 0) return
        if (list.scrollLeft < map * 0.5) list.scrollLeft += map
        else if (list.scrollLeft > map * 1.5) list.scrollLeft -= map
    }
    const drag = useDrag(scroller)

    return <div ref={scroller} className="season-map-heat" onScroll={wrap} {...drag}
                role="img" aria-label={t("archive.heat")}>
        {copies.map((copy, i) => <canvas key={i} ref={copy} width={HEAT_WIDTH} height={HEAT_HEIGHT}/>)}
    </div>
}

function useDrag(scroller: React.RefObject<HTMLDivElement>) {
    const from = useRef<{x: number, left: number}>()
    return {
        onPointerDown: (event: React.PointerEvent<HTMLDivElement>) => {
            from.current = {x: event.clientX, left: scroller.current?.scrollLeft ?? 0}
            event.currentTarget.setPointerCapture(event.pointerId)
        },
        onPointerMove: (event: React.PointerEvent<HTMLDivElement>) => {
            if (from.current && scroller.current) scroller.current.scrollLeft = from.current.left - (event.clientX - from.current.x)
        },
        onPointerUp: () => {
            from.current = undefined
        },
    }
}
