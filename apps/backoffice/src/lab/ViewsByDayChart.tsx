import {compactOf} from "../home/activity.ts"
import {useWidth} from "../home/useWidth.ts"
import {Dot, everyOf, labelled, logShareOf, pointedOf} from "./chart.ts"
import {dayLabelOf} from "./format.ts"
import {PostDots} from "./PostDots.tsx"
import {dayOf, Post, youngOf} from "./readout.ts"

const DAY = 86_400_000
const PLOT = 160
const TOP = 10
const AXIS = 22
const LEFT = 40
const RIGHT = 12

type Props = {
    posts: Post[]
    since: string
    today: string
    ticks: number[]
    breakout: number
    now: Date
    colorOf: (experiment: string) => string
    shown: string | undefined
    onHover: (postId: string | undefined) => void
    onPick: (postId: string | undefined) => void
}

export function ViewsByDayChart({posts, since, today, ticks, breakout, now, colorOf, shown, onHover, onPick}: Props) {
    const [box, width] = useWidth<HTMLDivElement>()
    const start = Date.parse(`${since}T00:00:00Z`)
    const days = Math.round((Date.parse(`${today}T00:00:00Z`) - start) / DAY) + 1
    const plot = Math.max(width - LEFT - RIGHT, 1)
    const band = plot / days
    const every = everyOf(band, 48)
    const x = (at: number) => LEFT + (at - start) / (days * DAY) * plot
    const y = (views: number) => TOP + PLOT * (1 - logShareOf(views, ticks))

    const dots: Dot[] = posts.map(post => ({
        key: post.postId, post, x: x(post.publishedAt.getTime()), y: y(post.views), color: colorOf(post.experiment),
        young: youngOf(post, now),
    }))

    return (
        <figure className="chart">
            <figcaption className="chart-title">Views by day</figcaption>
            <div ref={box} className="chart-box" onPointerMove={event => onHover(pointedOf(event, dots))}
                 onPointerLeave={() => onHover(undefined)} onPointerUp={event => onPick(pointedOf(event, dots))}>
                <svg width={width} height={TOP + PLOT + AXIS} role="img"
                     aria-label={`Views by day: ${posts.length} posts`}>
                    {ticks.map(tick => (
                        <g key={tick}>
                            <line className="chart-grid" x1={LEFT} x2={width - RIGHT} y1={y(tick)} y2={y(tick)}/>
                            <text className="chart-label" x={LEFT - 8} y={y(tick)} dy="0.32em" textAnchor="end">
                                {compactOf(tick)}
                            </text>
                        </g>
                    ))}
                    <line className="lab-breakout" x1={LEFT} x2={width - RIGHT} y1={y(breakout)} y2={y(breakout)}/>
                    {!ticks.includes(breakout) && (
                        <text className="chart-label" x={width - RIGHT} y={y(breakout) - 5} textAnchor="end">
                            {compactOf(breakout)}
                        </text>
                    )}
                    {Array.from({length: days}, (_, i) => labelled(i, days, every) && (
                        <text key={i} className="chart-label" x={LEFT + (i + 0.5) * band} y={TOP + PLOT + 16} textAnchor="middle">
                            {dayLabelOf(dayOf(new Date(start + i * DAY)))}
                        </text>
                    ))}
                    <PostDots dots={dots} shown={shown}/>
                </svg>
            </div>
        </figure>
    )
}
