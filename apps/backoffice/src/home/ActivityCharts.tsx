import {PointerEvent, useState} from "react"
import {
    compactOf,
    labelEveryOf,
    labelled,
    Slot,
    slotLabelOf,
    Span,
    tickLabelOf,
    ticksOf,
    totalsOf,
    wholeOf,
} from "./activity.ts"
import {useWidth} from "./useWidth.ts"

const PLOT = 120
const TOP = 18
const AXIS = 22
const LEFT = 44
const RIGHT = 10

type Props = {slots: Slot[], step: number, span: Span, players: number}

// Two charts of one series each, not one with two scales: a hover on either reads the same slot on both.
export function ActivityCharts({slots, step, span, players}: Props) {
    const [hovered, setHovered] = useState<number | undefined>(undefined)
    const totals = totalsOf(slots)
    const shown = hovered === undefined ? undefined : slots[hovered]
    const every = labelEveryOf(step)

    return (
        <section className="panel" aria-label="Activity">
            <p className="readout" aria-live="polite">
                {shown === undefined
                    ? <>Last {span.label}: <b>{wholeOf(totals.clicks)}</b> clicks, <b>{wholeOf(totals.tiles)}</b> tiles
                        changed hands, <b>{wholeOf(players)}</b> players</>
                    : <>{slotLabelOf(shown)}{shown.filling ? " (so far)" : ""}: <b>{wholeOf(shown.clicks)}</b> clicks,{" "}
                        <b>{wholeOf(shown.tiles)}</b> tiles, <b>{wholeOf(shown.players)}</b> players</>}
            </p>
            <SlotChart title="Clicks" slots={slots} valueOf={slot => slot.clicks} every={every}
                       hovered={hovered} onHover={setHovered}/>
            <SlotChart title="Players taking tiles" slots={slots} valueOf={slot => slot.players} every={every}
                       hovered={hovered} onHover={setHovered}/>
            <details className="chart-table">
                <summary>Table</summary>
                <SlotTable slots={slots}/>
            </details>
        </section>
    )
}

type ChartProps = {
    title: string
    slots: Slot[]
    valueOf: (slot: Slot) => number
    every: number
    hovered: number | undefined
    onHover: (slot: number | undefined) => void
}

function SlotChart({title, slots, valueOf, every, hovered, onHover}: ChartProps) {
    const [box, width] = useWidth<HTMLDivElement>()
    const values = slots.map(valueOf)
    const ticks = ticksOf(Math.max(0, ...values))
    const top = ticks[ticks.length - 1]
    const band = (width - LEFT - RIGHT) / Math.max(slots.length, 1)
    const x = (i: number) => LEFT + (i + 0.5) * band
    const y = (value: number) => TOP + PLOT * (1 - value / top)

    const settled = slots.length > 0 && slots[slots.length - 1].filling ? slots.length - 1 : slots.length
    const points = values.slice(0, settled).map((value, i) => `${x(i)},${y(value)}`)
    const peak = values.indexOf(Math.max(...values))

    const hover = (event: PointerEvent<HTMLDivElement>) => {
        const left = event.currentTarget.getBoundingClientRect().left
        const slot = Math.floor((event.clientX - left - LEFT) / band)
        onHover(slots.length === 0 ? undefined : Math.min(Math.max(slot, 0), slots.length - 1))
    }

    return (
        <figure className="chart">
            <figcaption className="chart-title">{title}</figcaption>
            <div ref={box} className="chart-box" onPointerMove={hover} onPointerLeave={() => onHover(undefined)}>
                <svg width={width} height={TOP + PLOT + AXIS} role="img"
                     aria-label={`${title}: at most ${wholeOf(Math.max(0, ...values))} a slot`}>
                    {ticks.map(tick => (
                        <g key={tick}>
                            <line className="chart-grid" x1={LEFT} x2={width - RIGHT} y1={y(tick)} y2={y(tick)}/>
                            <text className="chart-label" x={LEFT - 8} y={y(tick)} dy="0.32em" textAnchor="end">
                                {compactOf(tick)}
                            </text>
                        </g>
                    ))}
                    {slots.map((slot, i) => labelled(slot, every) && (
                        <text key={i} className="chart-label" x={LEFT + i * band} y={TOP + PLOT + 16} textAnchor="middle">
                            {tickLabelOf(slot, every)}
                        </text>
                    ))}
                    {points.length > 1 && (
                        <>
                            <path className="chart-area"
                                  d={`M${x(0)},${y(0)} L${points.join(" L")} L${x(settled - 1)},${y(0)} Z`}/>
                            <polyline className="chart-line" points={points.join(" ")}/>
                        </>
                    )}
                    {settled < slots.length && (
                        <circle className="chart-filling" cx={x(settled)} cy={y(values[settled])} r={3}>
                            <title>Still filling</title>
                        </circle>
                    )}
                    {peak >= 0 && values[peak] > 0 && hovered === undefined && (
                        <text className="chart-peak" x={x(peak)} y={y(values[peak]) - 8}
                              textAnchor={x(peak) < LEFT + 30 ? "start" : x(peak) > width - 40 ? "end" : "middle"}>
                            {compactOf(values[peak])}
                        </text>
                    )}
                    {hovered !== undefined && (
                        <>
                            <line className="chart-cross" x1={x(hovered)} x2={x(hovered)} y1={TOP} y2={TOP + PLOT}/>
                            <circle className="chart-dot" cx={x(hovered)} cy={y(values[hovered])} r={4}/>
                        </>
                    )}
                </svg>
            </div>
        </figure>
    )
}

function SlotTable({slots}: {slots: Slot[]}) {
    return (
        <table className="table">
            <thead>
                <tr>
                    <th scope="col">Time</th>
                    <th scope="col" className="num">Clicks</th>
                    <th scope="col" className="num">Tiles</th>
                    <th scope="col" className="num">Players</th>
                </tr>
            </thead>
            <tbody>
                {slots.map(slot => (
                    <tr key={slot.start.getTime()}>
                        <td>{slotLabelOf(slot)}</td>
                        <td className="num">{wholeOf(slot.clicks)}</td>
                        <td className="num">{wholeOf(slot.tiles)}</td>
                        <td className="num">{wholeOf(slot.players)}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}
