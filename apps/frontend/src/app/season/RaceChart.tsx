import {CSSProperties, useEffect, useState} from "react"
import {BookStep} from "../../backends/standings.ts"
import {Countries} from "../../domain/countries.ts"
import {headsOf, RaceMeasure, raceOf} from "../../domain/seasonStats.ts"
import Bubble from "../components/Bubble.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {PlayIcon} from "../components/icons.tsx"
import "./RaceChart.css"

const DURATION_MS = 1000
const WIDTH = 600
const HEIGHT = 280
const PAD = {left: 8, right: 44, top: 14, bottom: 8}
// The least room between two flags at the lines' ends, so none hides another.
const HEAD_GAP = 17

type Pointed = {countryCode: string, step: number, anchor: HTMLElement}

export default function RaceChart({steps, active}: {steps: BookStep[], active: boolean}) {
    const [measure, setMeasure] = useState<RaceMeasure>("points")
    const [replays, setReplays] = useState(0)
    const [focus, setFocus] = useState<string>()
    const [pointed, setPointed] = useState<Pointed>()
    const progress = useProgress(active, `${measure}:${replays}`)
    const {lines, most} = raceOf(steps, measure)
    if (steps.length === 0) return <p className="season-stats-none">No round scored</p>

    const at = progress * (steps.length - 1)
    const x = (i: number) => PAD.left + (steps.length > 1 ? i / (steps.length - 1) : 1) * (WIDTH - PAD.left - PAD.right)
    const y = (value: number) => HEIGHT - PAD.bottom - (value / most) * (HEIGHT - PAD.top - PAD.bottom)
    const hue = new Map(lines.map((line, i) => [line.countryCode, hueOf(i, lines.length)]))
    const heads = headsOf(lines.map((line) => ({countryCode: line.countryCode, y: y(valueAt(line.values, at))})), HEAD_GAP, PAD.top, HEIGHT - PAD.bottom)
    const ranked = lines.map((line) => ({...line, value: valueAt(line.values, at)})).sort((a, b) => b.value - a.value)
    const shown = focus ?? pointed?.countryCode
    const lineClass = (code: string) => shown === undefined ? "race-line" : shown === code ? "race-line race-line--on" : "race-line race-line--off"

    return <div className="race">
        <div className="race-bar">
            <div className="race-switch" role="radiogroup" aria-label="Measure">
                {([["points", "Points"], ["share", "Map share"]] as const).map(([value, name]) => <button key={value}
                                                                                                        type="button"
                                                                                                        role="radio"
                                                                                                        aria-checked={measure === value}
                                                                                                        className={measure === value ? "race-choice race-choice--on" : "race-choice"}
                                                                                                        onClick={() => setMeasure(value)}>{name}</button>)}
            </div>
            <span className="race-day">{whenLine(steps[Math.min(steps.length - 1, Math.round(at))].endedAt)}</span>
            <button type="button" className="icon-button race-replay" aria-label="Play again" onClick={() => setReplays((n) => n + 1)}>
                <PlayIcon size={16}/>
            </button>
        </div>

        <div className="race-plot" onMouseLeave={() => setPointed(undefined)}>
            <svg className="race-chart" viewBox={`0 0 ${WIDTH} ${HEIGHT}`} aria-hidden="true">
                {[0.25, 0.5, 0.75, 1].map((share) => <line key={share} className="race-grid" x1={PAD.left} x2={WIDTH - PAD.right}
                                                          y1={y(most * share)} y2={y(most * share)}/>)}
                {lines.map((line) => <polyline key={line.countryCode}
                                               className={lineClass(line.countryCode)}
                                               style={{"--author-hue": hue.get(line.countryCode)} as CSSProperties}
                                               points={pointsUpTo(line.values, at).map(([step, value]) => `${x(step)},${y(value)}`).join(" ")}/>)}
                {pointed && <circle className="race-dot" cx={x(pointed.step)} r={4.5}
                                    cy={y(lines.find((line) => line.countryCode === pointed.countryCode)?.values[pointed.step] ?? 0)}
                                    style={{"--author-hue": hue.get(pointed.countryCode)} as CSSProperties}/>}
                {heads.map((head) => {
                    const line = lines.find((kept) => kept.countryCode === head.countryCode)
                    const end = y(valueAt(line?.values ?? [], at))
                    return <g key={head.countryCode} className={lineClass(head.countryCode)}>
                        {Math.abs(head.y - end) > 1 && <line className="race-leader-line" x1={x(at)} y1={end} x2={x(at) + 7} y2={head.y}/>}
                        <circle className="race-head-dot" cx={x(at) + 9} cy={head.y} r={3.5}
                                style={{"--author-hue": hue.get(head.countryCode)} as CSSProperties}/>
                        <foreignObject x={x(at) + 14} y={head.y - 8} width={26} height={16}>
                            <span className="race-head"><CountryFlag code={head.countryCode}/></span>
                        </foreignObject>
                    </g>
                })}
            </svg>
            {progress === 1 && lines.flatMap((line) => line.values.map((value, step) => <button
                key={`${line.countryCode}:${step}`}
                type="button"
                className="race-point"
                style={{left: `${(x(step) / WIDTH) * 100}%`, top: `${(y(value) / HEIGHT) * 100}%`}}
                aria-label={pointLine(line.countryCode, steps[step].endedAt, value, step > 0 ? value - line.values[step - 1] : value, measure)}
                onMouseEnter={(e) => setPointed({countryCode: line.countryCode, step, anchor: e.currentTarget})}
                onFocus={(e) => setPointed({countryCode: line.countryCode, step, anchor: e.currentTarget})}
                onBlur={() => setPointed(undefined)}/>))}
        </div>

        <ol className="race-legend">
            {ranked.map((line, rank) => <li key={line.countryCode}
                                            className={shown === undefined || shown === line.countryCode ? "race-key" : "race-key race-key--off"}
                                            style={{"--author-hue": hue.get(line.countryCode)} as CSSProperties}
                                            onMouseEnter={() => setFocus(line.countryCode)}
                                            onMouseLeave={() => setFocus(undefined)}>
                <span className="race-key-rank">{rank + 1}</span>
                <span className="race-key-swatch" aria-hidden="true"/>
                <CountryFlag code={line.countryCode}/>
                <span className="race-key-name">{nameOf(line.countryCode)}</span>
                <span className="race-key-value">{valueLine(line.value, measure)}</span>
            </li>)}
        </ol>

        {pointed && <Bubble anchor={{current: pointed.anchor}}>
            <PointBubble countryCode={pointed.countryCode} step={steps[pointed.step]} measure={measure}
                         value={lines.find((line) => line.countryCode === pointed.countryCode)?.values[pointed.step] ?? 0}
                         gain={gainOf(lines.find((line) => line.countryCode === pointed.countryCode)?.values ?? [], pointed.step)}/>
        </Bubble>}
    </div>
}

function PointBubble({countryCode, step, measure, value, gain}: {countryCode: string, step: BookStep, measure: RaceMeasure, value: number, gain: number}) {
    return <span className="race-bubble">
        <span className="race-bubble-who"><CountryFlag code={countryCode}/>{nameOf(countryCode)}</span>
        <span>{whenLine(step.endedAt)}</span>
        <span className="race-bubble-value">{valueLine(value, measure)}{measure === "points" && gain > 0 ? ` (+${gain})` : ""}</span>
    </span>
}

function pointLine(code: string, at: number, value: number, gain: number, measure: RaceMeasure): string {
    return `${nameOf(code)}, ${whenLine(at)}: ${valueLine(value, measure)}${measure === "points" && gain > 0 ? `, +${gain}` : ""}`
}

function valueLine(value: number, measure: RaceMeasure): string {
    return measure === "points" ? `${Math.round(value)} points` : `${(value * 100).toFixed(1)}% of the map`
}

function gainOf(values: number[], step: number): number {
    return step > 0 ? values[step] - values[step - 1] : values[step] ?? 0
}

function nameOf(code: string): string {
    return Countries.get(code)?.name ?? code
}

// "Sat 31 Oct, 23:00", in the player's time.
function whenLine(at: number): string {
    const parts = new Intl.DateTimeFormat("en-GB", {
        weekday: "short", day: "numeric", month: "short", hour: "2-digit", minute: "2-digit", hourCycle: "h23",
    }).formatToParts(at)
    const part = (type: Intl.DateTimeFormatPartTypes) => parts.find((p) => p.type === type)?.value ?? ""
    return `${part("weekday")} ${part("day")} ${part("month")}, ${part("hour")}:${part("minute")}`
}

// The line from its first round to a fractional one, the last point between two rounds.
function pointsUpTo(values: number[], at: number): [number, number][] {
    const whole = Math.floor(at)
    const points: [number, number][] = values.slice(0, whole + 1).map((value, i) => [i, value])
    if (at > whole && whole + 1 < values.length) points.push([at, valueAt(values, at)])
    return points
}

function valueAt(values: number[], at: number): number {
    const whole = Math.floor(at)
    const next = values[Math.min(values.length - 1, whole + 1)] ?? 0
    return (values[whole] ?? 0) + (next - (values[whole] ?? 0)) * (at - whole)
}

function hueOf(i: number, count: number): number {
    return Math.round((i * 360) / Math.max(1, count) + 20) % 360
}

// From 0 to 1 over the chart's run, each time it shows or a key changes; at once under reduced motion.
function useProgress(active: boolean, key: string): number {
    const [progress, setProgress] = useState(0)

    useEffect(() => {
        if (!active) return
        if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) {
            setProgress(1)
            return
        }
        let frame = 0
        const start = performance.now()
        const tick = (now: number) => {
            const done = Math.min(1, (now - start) / DURATION_MS)
            setProgress(done)
            if (done < 1) frame = requestAnimationFrame(tick)
        }
        frame = requestAnimationFrame(tick)
        return () => cancelAnimationFrame(frame)
    }, [active, key])

    return progress
}
