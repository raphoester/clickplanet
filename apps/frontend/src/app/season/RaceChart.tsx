import {CSSProperties, useEffect, useState} from "react"
import {BookStep} from "../../backends/standings.ts"
import {Countries} from "../../domain/countries.ts"
import {RaceMeasure, raceOf} from "../../domain/seasonStats.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import {PlayIcon} from "../components/icons.tsx"
import "./RaceChart.css"

const DURATION_MS = 5000
const WIDTH = 600
const HEIGHT = 280
const PAD = {left: 8, right: 40, top: 14, bottom: 8}

export default function RaceChart({steps, active}: {steps: BookStep[], active: boolean}) {
    const [measure, setMeasure] = useState<RaceMeasure>("points")
    const [replays, setReplays] = useState(0)
    const progress = useProgress(active, `${measure}:${replays}`)
    const {lines, most} = raceOf(steps, measure)
    if (steps.length === 0) return <p className="season-stats-none">No round scored</p>

    const at = progress * (steps.length - 1)
    const x = (i: number) => PAD.left + (steps.length > 1 ? i / (steps.length - 1) : 1) * (WIDTH - PAD.left - PAD.right)
    const y = (value: number) => HEIGHT - PAD.bottom - (value / most) * (HEIGHT - PAD.top - PAD.bottom)
    const heads = lines.map((line) => ({...line, value: valueAt(line.values, at)})).sort((a, b) => b.value - a.value)
    const leader = heads[0]

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
            <span className="race-day">Day {Math.min(steps.length, Math.floor(at) + 1)} of {steps.length}</span>
            <button type="button" className="icon-button race-replay" aria-label="Play again" onClick={() => setReplays((n) => n + 1)}>
                <PlayIcon size={16}/>
            </button>
        </div>
        <svg className="race-chart" viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role="img"
             aria-label={leader ? `${Countries.get(leader.countryCode)?.name ?? leader.countryCode} leads` : "The race"}>
            {[0.25, 0.5, 0.75, 1].map((share) => <line key={share} className="race-grid" x1={PAD.left} x2={WIDTH - PAD.right}
                                                      y1={y(most * share)} y2={y(most * share)}/>)}
            {lines.map((line, i) => <polyline key={line.countryCode}
                                              className="race-line"
                                              style={{"--author-hue": hueOf(i, lines.length)} as CSSProperties}
                                              points={pointsUpTo(line.values, at).map(([step, value]) => `${x(step)},${y(value)}`).join(" ")}/>)}
            {heads.map((head) => <foreignObject key={head.countryCode} x={x(at) + 3} y={y(head.value) - 9} width={30} height={18}>
                <span className="race-head"><CountryFlag code={head.countryCode}/></span>
            </foreignObject>)}
        </svg>
        {leader && <p className="race-leader">
            <CountryFlag code={leader.countryCode}/>
            <span>{Countries.get(leader.countryCode)?.name ?? leader.countryCode}</span>
        </p>}
    </div>
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
