import {CSSProperties} from "react"
import {useWidth} from "../home/useWidth.ts"
import {rightBarOf} from "./chart.ts"
import {count, percent, rate, secondsOf} from "./format.ts"
import {Arm, Platform, Reading} from "./readout.ts"

const BAR = 12
const PAIR = 2
const GAP = 10

type Sign = {
    title: string
    valueOf: (arm: Arm) => number | undefined
    labelOf: (value: number | undefined) => string
    noteOf?: (arm: Arm) => string
}

const POSTS: Sign = {title: "Posts", valueOf: arm => arm.posts.length, labelOf: count}

const SIGNS: Record<Platform, Sign[]> = {
    tiktok: [
        {...POSTS, noteOf: arm => {
            const waiting = arm.posts.length - arm.business.posts
            return waiting > 0 ? `(${waiting} too young)` : ""
        }},
        {title: "Held at 2 s", valueOf: arm => arm.business.heldAt2s, labelOf: percent},
        {title: "To the end", valueOf: arm => arm.business.toTheEnd, labelOf: percent},
        {title: "Average watch", valueOf: arm => arm.business.averageWatchSeconds, labelOf: secondsOf},
        {title: "Comments /1k", valueOf: arm => arm.perThousand.comments, labelOf: rate},
        {title: "Shares /1k", valueOf: arm => arm.perThousand.shares, labelOf: rate},
        {title: "Follows /1k", valueOf: arm => arm.perThousand.follows, labelOf: rate},
        {title: "Saves /1k", valueOf: arm => arm.perThousand.saves, labelOf: rate},
    ],
    youtube: [
        POSTS,
        {title: "Likes /1k", valueOf: arm => arm.perThousand.likes, labelOf: rate},
    ],
}

type Props = {platform: Platform, readings: Reading[], colorOf: (experiment: string) => string}

export function SignsChart({platform, readings, colorOf}: Props) {
    return (
        <div className="lab-signs">
            {SIGNS[platform].map(sign => <SignPanel key={sign.title} sign={sign} readings={readings} colorOf={colorOf}/>)}
        </div>
    )
}

function SignPanel({sign, readings, colorOf}: {sign: Sign, readings: Reading[], colorOf: (experiment: string) => string}) {
    const [box, width] = useWidth<HTMLDivElement>()
    const arms = readings.flatMap(reading => [reading.experiment, reading.control])
    const values = arms.map(sign.valueOf)
    const labels = values.map(sign.labelOf)
    const notes = arms.map(arm => sign.noteOf?.(arm) ?? "")
    const longest = Math.max(...labels.map((label, i) => label.length + (notes[i] === "" ? 0 : notes[i].length + 1)))
    const room = Math.min(width * 0.7, 12 + 7.5 * longest)
    const top = Math.max(0, ...values.map(value => value ?? 0))
    const lengthOf = (value: number | undefined) => value === undefined || top === 0 ? 0 : (width - room) * value / top
    const yOf = (i: number) => Math.floor(i / 2) * (2 * BAR + PAIR + GAP) + (i % 2) * (BAR + PAIR)
    const height = readings.length * (2 * BAR + PAIR + GAP) - GAP
    const said = arms.map((arm, i) => [arm.experiment, labels[i], notes[i]].filter(part => part !== "").join(" "))

    return (
        <figure className="chart">
            <figcaption className="chart-title">{sign.title}</figcaption>
            <div ref={box}>
                <svg width={width} height={height} role="img"
                     aria-label={`${sign.title}: ${said.join(", ")}`}>
                    <line className="chart-grid" x1={0.5} x2={0.5} y1={0} y2={height}/>
                    {arms.map((arm, i) => {
                        const length = lengthOf(values[i])
                        return (
                            <g key={i} style={{"--arm": colorOf(arm.experiment)} as CSSProperties}>
                                {length > 0 && <path className="lab-bar" d={rightBarOf({x: 0, y: yOf(i), length, thickness: BAR})}/>}
                                <text className="lab-value" x={length + 6} y={yOf(i) + BAR / 2} dy="0.32em">
                                    {labels[i]}
                                    {notes[i] !== "" && <tspan className="lab-value-note"> {notes[i]}</tspan>}
                                </text>
                            </g>
                        )
                    })}
                </svg>
            </div>
        </figure>
    )
}
