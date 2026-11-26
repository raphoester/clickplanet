import {CSSProperties} from "react"

export type Mark = "dot" | "young" | "bar" | "filling" | "median" | "breakout"

export type Key = {label: string, mark: Mark, color?: string}

const MUTED = "var(--muted)"

export function Legend({keys}: {keys: Key[]}) {
    return (
        <ul className="lab-legend" aria-label="Legend">
            {keys.map(key => (
                <li key={key.label} className={key.color === undefined ? "lab-key" : undefined}>
                    <Swatch mark={key.mark} color={key.color ?? MUTED}/>
                    {key.label}
                </li>
            ))}
        </ul>
    )
}

function Swatch({mark, color}: {mark: Mark, color: string}) {
    const style = {"--arm": color} as CSSProperties
    return (
        <svg width={12} height={12} aria-hidden="true" style={style}>
            {mark === "dot" && <circle className="lab-dot" cx={6} cy={6} r={4}/>}
            {mark === "young" && <circle className="lab-dot-young" cx={6} cy={6} r={4}/>}
            {mark === "bar" && <rect className="lab-bar" x={1} y={1} width={10} height={10} rx={2}/>}
            {mark === "filling" && <rect className="lab-bar-filling" x={1.5} y={1.5} width={9} height={9} rx={2}/>}
            {mark === "median" && <line className="lab-median" x1={6} x2={6} y1={1} y2={11}/>}
            {mark === "breakout" && <line className="lab-breakout" x1={6} x2={6} y1={0} y2={12}/>}
        </svg>
    )
}
