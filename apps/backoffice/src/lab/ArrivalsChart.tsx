import {CSSProperties, PointerEvent, useState} from "react"
import {ticksOf, wholeOf} from "../home/activity.ts"
import {useWidth} from "../home/useWidth.ts"
import {arrivalBarsOf, everyOf, labelled, upBarOf} from "./chart.ts"
import {dayLabelOf} from "./format.ts"
import {Legend} from "./Legend.tsx"
import {ArrivalDay, arrivalTotalsOf} from "./readout.ts"

const PLOT = 120
const TOP = 10
const AXIS = 22
const LEFT = 30
const RIGHT = 8
const SOURCES = [
    {key: "tiktok", label: "TikTok (/t)", color: "var(--lab-tiktok)"},
    {key: "youtube", label: "YouTube (/y)", color: "var(--lab-youtube)"},
] as const

export function ArrivalsChart({days, today}: {days: ArrivalDay[], today: string}) {
    const [hovered, setHovered] = useState<number | undefined>(undefined)
    const [box, width] = useWidth<HTMLDivElement>()
    const bars = arrivalBarsOf(days, today)
    const ticks = ticksOf(Math.max(0, ...bars.flatMap(bar => [bar.tiktok, bar.youtube])))
    const top = ticks[ticks.length - 1]
    const plot = Math.max(width - LEFT - RIGHT, 1)
    const band = plot / Math.max(bars.length, 1)
    const thickness = Math.max(2, Math.min(12, (band * 0.7 - 2) / 2))
    const every = everyOf(band, 48)
    const y = (value: number) => TOP + PLOT * (1 - value / top)
    const shown = hovered === undefined ? undefined : bars[hovered]
    const totals = arrivalTotalsOf(bars)

    const hover = (event: PointerEvent<HTMLDivElement>) => {
        const left = event.currentTarget.getBoundingClientRect().left
        const slot = Math.floor((event.clientX - left - LEFT) / band)
        setHovered(bars.length === 0 ? undefined : Math.min(Math.max(slot, 0), bars.length - 1))
    }

    return (
        <>
            <Legend keys={[...SOURCES.map(source => ({label: source.label, mark: "bar" as const, color: source.color})),
                {label: "Today", mark: "filling"}]}/>
            <p className="readout" aria-live="polite">
                {shown === undefined
                    ? bars.length > 0 && <>Since {dayLabelOf(bars[0].day)}: <b>{wholeOf(totals.tiktok)}</b> from TikTok,{" "}
                        <b>{wholeOf(totals.youtube)}</b> from YouTube</>
                    : <>{dayLabelOf(shown.day)}{shown.filling ? " (so far)" : ""}: <b>{wholeOf(shown.tiktok)}</b> from
                        TikTok, <b>{wholeOf(shown.youtube)}</b> from YouTube</>}
            </p>
            <figure className="chart">
                <div ref={box} className="chart-box" onPointerMove={hover} onPointerLeave={() => setHovered(undefined)}>
                    <svg width={width} height={TOP + PLOT + AXIS} role="img"
                         aria-label={`Arrivals: ${wholeOf(totals.tiktok)} from TikTok, ${wholeOf(totals.youtube)} from YouTube`}>
                        {hovered !== undefined && (
                            <rect className="lab-band" x={LEFT + hovered * band} y={TOP} width={band} height={PLOT}/>
                        )}
                        {ticks.map(tick => (
                            <g key={tick}>
                                <line className="chart-grid" x1={LEFT} x2={width - RIGHT} y1={y(tick)} y2={y(tick)}/>
                                <text className="chart-label" x={LEFT - 8} y={y(tick)} dy="0.32em" textAnchor="end">
                                    {wholeOf(tick)}
                                </text>
                            </g>
                        ))}
                        {bars.map((bar, i) => labelled(i, bars.length, every) && (
                            <text key={bar.day} className="chart-label" x={LEFT + (i + 0.5) * band} y={TOP + PLOT + 16}
                                  textAnchor="middle">
                                {dayLabelOf(bar.day)}
                            </text>
                        ))}
                        {bars.map((bar, i) => SOURCES.map((source, s) => {
                            const value = bar[source.key]
                            if (value === 0) return null
                            const x = LEFT + i * band + band / 2 - thickness - 1 + s * (thickness + 2)
                            return (
                                <path key={`${bar.day}-${source.key}`} className={bar.filling ? "lab-bar-filling" : "lab-bar"}
                                      style={{"--arm": source.color} as CSSProperties}
                                      d={upBarOf({x, y: y(0), length: y(0) - y(value), thickness})}/>
                            )
                        }))}
                    </svg>
                </div>
            </figure>
        </>
    )
}
