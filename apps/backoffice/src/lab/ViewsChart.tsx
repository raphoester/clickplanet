import {compactOf} from "../home/activity.ts"
import {useWidth} from "../home/useWidth.ts"
import {Dot, lanesOf, logShareOf, pointedOf} from "./chart.ts"
import {viewsOf} from "./format.ts"
import {PostDots} from "./PostDots.tsx"
import {Arm, Reading, youngOf} from "./readout.ts"

const ROW = 36
const GAP = 12
const TOP = 18
const AXIS = 22
const RIGHT = 48
const LANES = [0, -6, 6, -12, 12]

type Props = {
    readings: Reading[]
    ticks: number[]
    breakout: number
    now: Date
    colorOf: (experiment: string) => string
    shown: string | undefined
    onHover: (postId: string | undefined) => void
    onPick: (postId: string | undefined) => void
}

type Row = {arm: Arm, y: number, control: boolean}

export function ViewsChart({readings, ticks, breakout, now, colorOf, shown, onHover, onPick}: Props) {
    const [box, width] = useWidth<HTMLDivElement>()
    const rows: Row[] = readings.flatMap((reading, i) => [reading.experiment, reading.control].map((arm, j) => ({
        arm, y: TOP + i * (2 * ROW + GAP) + (j + 0.5) * ROW, control: j === 1,
    })))
    const longest = Math.max(...rows.map(row => row.arm.experiment.length))
    const left = Math.min(Math.round(width * 0.36), 14 + 7 * longest)
    const plot = Math.max(width - left - RIGHT, 1)
    const x = (views: number) => left + logShareOf(views, ticks) * plot
    const bottom = TOP + readings.length * (2 * ROW + GAP) - GAP

    const dots: Dot[] = rows.flatMap((row, r) => {
        const xs = row.arm.posts.map(post => x(post.views))
        const lanes = lanesOf(xs, 9, LANES.length)
        return row.arm.posts.map((post, i) => ({
            key: `${r}-${post.postId}`, post, x: xs[i], y: row.y + LANES[lanes[i]], color: colorOf(post.experiment),
            young: youngOf(post, now),
        }))
    })

    const label = readings.map(({experiment, control}) =>
        `${experiment.experiment} median ${viewsOf(experiment.medianViews)}, its control ${viewsOf(control.medianViews)}`,
    ).join("; ")

    return (
        <figure className="chart">
            <figcaption className="chart-title">Views</figcaption>
            <div ref={box} className="chart-box" onPointerMove={event => onHover(pointedOf(event, dots))}
                 onPointerLeave={() => onHover(undefined)} onPointerUp={event => onPick(pointedOf(event, dots))}>
                <svg width={width} height={bottom + AXIS} role="img" aria-label={`Views: ${label}`}>
                    {ticks.map(tick => (
                        <g key={tick}>
                            <line className="chart-grid" x1={x(tick)} x2={x(tick)} y1={TOP - 4} y2={bottom}/>
                            <text className="chart-label" x={x(tick)} y={bottom + 16} textAnchor="middle">{compactOf(tick)}</text>
                        </g>
                    ))}
                    {readings.slice(1).map((_, i) => {
                        const y = TOP + (i + 1) * (2 * ROW + GAP) - GAP / 2
                        return <line key={i} className="chart-grid" x1={0} x2={width} y1={y} y2={y}/>
                    })}
                    <line className="lab-breakout" x1={x(breakout)} x2={x(breakout)} y1={TOP - 6} y2={bottom}/>
                    <text className="chart-label" x={x(breakout)} y={TOP - 9} textAnchor="middle">{compactOf(breakout)}</text>
                    {rows.map((row, r) => (
                        <text key={r} className={row.control ? "lab-name lab-name-control" : "lab-name"}
                              x={left - 10} y={row.y} dy="0.32em" textAnchor="end">
                            {row.arm.experiment}
                        </text>
                    ))}
                    <PostDots dots={dots} shown={shown}/>
                    {rows.map((row, r) => row.arm.medianViews !== undefined && (
                        <g key={r}>
                            <line className="lab-median-halo" x1={x(row.arm.medianViews)} x2={x(row.arm.medianViews)}
                                  y1={row.y - 15} y2={row.y + 15}/>
                            <line className="lab-median" x1={x(row.arm.medianViews)} x2={x(row.arm.medianViews)}
                                  y1={row.y - 15} y2={row.y + 15}/>
                        </g>
                    ))}
                    {rows.map((row, r) => (
                        <text key={r} className="lab-value" x={width - RIGHT + 10} y={row.y} dy="0.32em">
                            {viewsOf(row.arm.medianViews)}
                        </text>
                    ))}
                </svg>
            </div>
        </figure>
    )
}
