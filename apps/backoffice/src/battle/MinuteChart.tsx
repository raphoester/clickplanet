import {PointerEvent} from "react"
import {compactOf, ticksOf, wholeOf} from "../home/activity.ts"
import {useWidth} from "../home/useWidth.ts"

const PLOT = 120
const TOP = 18
const AXIS = 22
const LEFT = 44
const RIGHT = 10
// The palette's slots, in order: a series keeps its slot whatever else is shown.
export const SLOTS = 5

const CLOCK = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

// A series has a value a minute, or none for a minute no reading spans.
export type Series = {key: string, label: string, slot: number, values: (number | undefined)[]}

type Props = {
    title: string
    starts: Date[]
    // Minutes between two time labels.
    every: number
    series: Series[]
    // The last minute is still filling: its dot stands apart, off the line.
    filling: boolean
    hovered: number | undefined
    onHover: (minute: number | undefined) => void
}

// Lines of one unit on one scale. One series needs no legend: the title names it.
export function MinuteChart({title, starts, every, series, filling, hovered, onHover}: Props) {
    const [box, width] = useWidth<HTMLDivElement>()
    const known = series.flatMap(line => line.values.filter((value): value is number => value !== undefined))
    const highest = Math.max(0, ...known)
    const ticks = ticksOf(highest)
    const top = ticks[ticks.length - 1]
    const band = (width - LEFT - RIGHT) / Math.max(starts.length, 1)
    const x = (i: number) => LEFT + (i + 0.5) * band
    const y = (value: number) => TOP + PLOT * (1 - value / top)
    const settled = filling ? starts.length - 1 : starts.length

    const hover = (event: PointerEvent<HTMLDivElement>) => {
        const left = event.currentTarget.getBoundingClientRect().left
        const minute = Math.floor((event.clientX - left - LEFT) / band)
        onHover(starts.length === 0 ? undefined : Math.min(Math.max(minute, 0), starts.length - 1))
    }

    return (
        <figure className="chart">
            <figcaption className="chart-title">{title}</figcaption>
            {series.length > 1 && (
                <ul className="legend">
                    {series.map(line => (
                        <li key={line.key}>
                            <span className={`swatch chart-series-${line.slot}`} aria-hidden="true"/>
                            {line.label}
                        </li>
                    ))}
                </ul>
            )}
            <div ref={box} className="chart-box" onPointerMove={hover} onPointerLeave={() => onHover(undefined)}>
                <svg width={width} height={TOP + PLOT + AXIS} role="img"
                     aria-label={`${title}: at most ${wholeOf(Math.round(highest))} a minute`}>
                    {ticks.map(tick => (
                        <g key={tick}>
                            <line className="chart-grid" x1={LEFT} x2={width - RIGHT} y1={y(tick)} y2={y(tick)}/>
                            <text className="chart-label" x={LEFT - 8} y={y(tick)} dy="0.32em" textAnchor="end">
                                {compactOf(tick)}
                            </text>
                        </g>
                    ))}
                    {starts.map((start, i) => start.getMinutes() % every === 0 && (
                        <text key={i} className="chart-label" x={LEFT + i * band} y={TOP + PLOT + 16} textAnchor="middle">
                            {CLOCK.format(start)}
                        </text>
                    ))}
                    {series.map(line => (
                        <g key={line.key} className={`chart-series-${line.slot}`}>
                            {runsOf(line.values.slice(0, settled)).map(run => run.length === 1
                                ? <circle key={run[0]} className="chart-point" cx={x(run[0])} cy={y(line.values[run[0]] ?? 0)} r={2}/>
                                : <polyline key={run[0]} className="chart-line"
                                            points={run.map(i => `${x(i)},${y(line.values[i] ?? 0)}`).join(" ")}/>)}
                            {filling && line.values[settled] !== undefined && (
                                <circle className="chart-filling" cx={x(settled)} cy={y(line.values[settled] ?? 0)} r={3}>
                                    <title>Still filling</title>
                                </circle>
                            )}
                        </g>
                    ))}
                    {hovered !== undefined && (
                        <>
                            <line className="chart-cross" x1={x(hovered)} x2={x(hovered)} y1={TOP} y2={TOP + PLOT}/>
                            {series.map(line => line.values[hovered] !== undefined && (
                                <circle key={line.key} className={`chart-dot chart-series-${line.slot}`}
                                        cx={x(hovered)} cy={y(line.values[hovered] ?? 0)} r={4}/>
                            ))}
                        </>
                    )}
                </svg>
            </div>
        </figure>
    )
}

// The runs of minutes with a value, each as the indexes it holds.
function runsOf(values: (number | undefined)[]): number[][] {
    const runs: number[][] = []
    values.forEach((value, i) => {
        if (value === undefined) return
        const run = runs[runs.length - 1]
        if (run !== undefined && run[run.length - 1] === i - 1) run.push(i)
        else runs.push([i])
    })
    return runs
}
