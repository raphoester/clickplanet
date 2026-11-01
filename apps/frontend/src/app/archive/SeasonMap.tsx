import {useEffect, useMemo, useRef, useState} from "react"
import {ReplayLandmasses} from "../../backends/replayBackend.ts"
import {HistoryBackend} from "../../backends/historyBackend.ts"
import {TurnBackend} from "../../backends/turn.ts"
import {Countries} from "../../domain/countries.ts"
import {LeaderboardEntry} from "../../domain/leaderboard.ts"
import {MapHistory, MapReel} from "../../domain/mapHistory.ts"
import {createGlobe} from "../viewer/globe.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {PauseIcon, PlayIcon} from "../components/icons.tsx"
import {drawHeat, lookOfThePage} from "./heatmap.ts"
import "./SeasonMap.css"

export type SeasonMapProps = {
    season: number
    turn: Pick<TurnBackend, "seasonReplay">
    geometry: () => Promise<{positions: Float32Array, landmasses: ReplayLandmasses}>
}

type Part = "season" | "finale"
type View = "replay" | "heat"

// Frames a second while it plays: a season of 30-minute frames in about 48 s, a finale of 10-second frames in 30 s.
const FPS = 24

export default function SeasonMap({season, turn, geometry}: SeasonMapProps) {
    const [part, setPart] = useState<Part>("season")
    const [view, setView] = useState<View>("replay")
    const [geometryRead, setGeometryRead] = useState<{positions: Float32Array, landmasses: ReplayLandmasses}>()
    const [read, setRead] = useState<{part: Part, history?: MapHistory}>()
    const history = read?.part === part ? read.history : undefined
    const loading = read?.part !== part

    useEffect(() => {
        void geometry().then(setGeometryRead)
    }, [geometry])

    useEffect(() => {
        let stale = false
        turn.seasonReplay(season, part === "finale").then(
            (answer) => {
                if (!stale) setRead({part, history: answer})
            },
            (e) => {
                console.error("Could not read the season's replay", e)
                if (!stale) setRead({part})
            },
        )
        return () => {
            stale = true
        }
    }, [turn, season, part])

    const reel = useMemo(() => history && geometryRead ? new MapReel(history, geometryRead.positions.length / 3) : undefined,
        [history, geometryRead])

    return <div className="season-map">
        <header className="panel season-map-head">
            <a className="button button-mini button-secondary" href="/play">Back to the game</a>
            <h1 className="season-map-title">Season {season}</h1>
            <Switch label="Window" value={part} onChange={setPart} choices={[["season", "Season"], ["finale", "Final Battle"]]}/>
            <Switch label="View" value={view} onChange={setView} choices={[["replay", "Replay"], ["heat", "Heatmap"]]}/>
        </header>

        <div className="season-map-stage">
            {loading && <p className="season-map-note">Rebuilding the map…</p>}
            {!loading && !history && <p className="season-map-note">No replay for this season</p>}
            {reel && geometryRead && view === "replay" && <Replay key={part} reel={reel} landmasses={geometryRead.landmasses}/>}
            {history && geometryRead && view === "heat" && <Heat positions={geometryRead.positions} hits={history.hits}/>}
        </div>
    </div>
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

function Replay({reel, landmasses}: {reel: MapReel, landmasses: ReplayLandmasses}) {
    const stage = useRef<HTMLDivElement>(null)
    const backend = useRef<HistoryBackend>()
    const [frame, setFrame] = useState(0)
    const [playing, setPlaying] = useState(false)
    const [leaders, setLeaders] = useState<LeaderboardEntry[]>([])
    const last = reel.length - 1

    useEffect(() => {
        const container = stage.current
        if (!container) return
        const aborted = new AbortController()
        const shown = new HistoryBackend(reel, landmasses)
        backend.current = shown
        const none = () => {}
        const made = createGlobe({
            tileClicker: {clickTile: () => Promise.resolve()},
            ownershipsGetter: shown,
            updatesListener: shown,
            container,
            country: {code: "", name: ""},
            mapView: "split",
            rendering: "sharp",
            onLeaderboardChange: (entries) => setLeaders(entries.slice(0, 5)),
            onLoadProgress: none, onRateLimited: none, onFrozen: none, onVPNBlocked: none, onBanned: none,
            onSessionUnavailable: none, onAllegianceLocked: none, onBonusTaken: none, onBonusWon: none, onCharges: none,
            onRules: none, onSwitchesChange: none, onBombDropped: none, onArmedChange: none,
            signal: aborted.signal,
        })
        return () => {
            aborted.abort()
            backend.current = undefined
            void made.then((globe) => globe.dispose())
        }
    }, [reel, landmasses])

    useEffect(() => {
        backend.current?.seek(frame)
    }, [frame])

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
        <ol className="panel season-map-leaders" aria-label="Most tiles held">
            {leaders.map((entry, i) => <li key={entry.country.code} className="season-map-leader">
                <span className="season-map-rank">{i + 1}</span>
                <CountryFlag code={entry.country.code}/>
                <span className="season-map-leader-name">{Countries.get(entry.country.code)?.name ?? entry.country.code}</span>
                <span className="season-map-leader-tiles">{entry.tiles.toLocaleString("en-US")}</span>
            </li>)}
        </ol>
        <footer className="panel season-map-controls">
            <button type="button"
                    className="icon-button"
                    aria-label={playing ? "Pause" : "Play"}
                    onClick={() => {
                        if (!playing && frame >= last) setFrame(0)
                        setPlaying(!playing)
                    }}>
                {playing ? <PauseIcon/> : <PlayIcon/>}
            </button>
            <input className="season-map-cursor"
                   type="range"
                   min={0}
                   max={last}
                   value={frame}
                   aria-label="Time"
                   onChange={(e) => {
                       setPlaying(false)
                       setFrame(Number(e.target.value))
                   }}/>
            <span className="season-map-time">{timeLine(reel.atOf(frame))}</span>
        </footer>
    </>
}

function Heat({positions, hits}: {positions: Float32Array, hits: ReadonlyMap<number, number>}) {
    const canvas = useRef<HTMLCanvasElement>(null)

    useEffect(() => {
        if (canvas.current) drawHeat(canvas.current, positions, hits, lookOfThePage())
    }, [positions, hits])

    return <canvas ref={canvas} className="season-map-heat" width={2048} height={1024} aria-label="Where the tiles were fought over"/>
}

// "Sat 31 Oct · 21:34", in the player's time.
function timeLine(at: number): string {
    const parts = new Intl.DateTimeFormat("en-GB", {
        weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", hourCycle: "h23",
    }).formatToParts(at)
    const part = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value ?? ""
    return `${part("weekday")} ${part("day")} ${part("month")} · ${part("hour")}:${part("minute")}`
}
