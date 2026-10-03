import {ReactNode, useId} from "react"
import {PlayerTitle} from "../../backends/player.ts"
import {enamelOf, METALS, metalOf} from "./titleArt.ts"

const FLAME = "M50 26 C59 39 67 46 67 59 C67 70 59 76 50 76 C41 76 33 70 33 59 C33 50 39 45 41 37 C45 44 47 47 50 50 C51 41 48 34 50 26 Z"
const FLAME_INNER = "M50 50 C55 56 59 60 59 65 C59 71 55 74 50 74 C45 74 41 71 41 65 C41 60 45 57 50 50 Z"
const SMALLER = "translate(50 52) scale(0.72) translate(-50 -52)"
const BUBBLE = "M28 32 Q28 26 34 26 H66 Q72 26 72 32 V54 Q72 60 66 60 H48 L37 71 V60 H34 Q28 60 28 54 Z"

const TICKS = Array.from({length: 24}, (_, i) => {
    const angle = i * 2 * Math.PI / 24
    return {x: 50 + 42 * Math.cos(angle), y: 50 + 42 * Math.sin(angle)}
})

function icon(title: PlayerTitle, ink: string, accent: string): ReactNode {
    switch (title.id) {
        case "og":
            return <text x="50" y="62" textAnchor="middle" fontFamily="Luckiest Guy, sans-serif" fontSize="34"
                         fill={accent} stroke="#1F1238" strokeWidth="1">OG</text>
        case "settler":
            return <>
                <path d="M42 71 V29" stroke={ink} strokeWidth="5" strokeLinecap="round" fill="none"/>
                <path d="M44 30 L68 37 L44 45 Z" fill={accent}/>
                <path d="M31 71 H61" stroke={ink} strokeWidth="5" strokeLinecap="round" fill="none"/>
            </>
        case "raider":
            return <>
                <path d="M36 70 L62 34" stroke={ink} strokeWidth="5" strokeLinecap="round" fill="none"/>
                <path d="M56 28 C68 28 74 38 72 48 L58 40 Z" fill={accent}/>
                <path d="M56 28 L52 44 L58 40 Z" fill={accent}/>
            </>
        case "warlord":
            return <>
                <path d="M32 62 C32 44 40 36 50 36 C60 36 68 44 68 62 L58 62 L58 54 L42 54 L42 62 Z" fill={accent}/>
                <path d="M50 54 V66" stroke={ink} strokeWidth="4" strokeLinecap="round" fill="none"/>
                <path d="M34 46 C26 42 24 34 26 27 C29 33 33 37 38 39" stroke={ink} strokeWidth="4" strokeLinecap="round" fill="none"/>
                <path d="M66 46 C74 42 76 34 74 27 C71 33 67 37 62 39" stroke={ink} strokeWidth="4" strokeLinecap="round" fill="none"/>
            </>
        case "conqueror":
            return <>
                <path d="M33 67 L66 34 M67 67 L34 34" stroke={ink} strokeWidth="5" strokeLinecap="round" fill="none"/>
                <path d="M34 55 L45 66 M66 55 L55 66" stroke={accent} strokeWidth="5" strokeLinecap="round" fill="none"/>
            </>
        case "warmaster":
            return <>
                <path d="M44 76 V30" stroke={ink} strokeWidth="4.5" strokeLinecap="round" fill="none"/>
                <path d="M44 18 L50 30 L38 30 Z" fill={ink}/>
                <path d="M46 32 H72 L64 42 L72 52 H46 Z" fill={accent}/>
                <path d="M55 37 L56.8 40.8 L61 41.2 L57.8 44 L58.8 48 L55 45.8 L51.2 48 L52.2 44 L49 41.2 L53.2 40.8 Z" fill={ink}/>
                <path d="M34 76 H54" stroke={ink} strokeWidth="4.5" strokeLinecap="round" fill="none"/>
            </>
        case "loyal":
            return <path d={FLAME} fill={accent} transform="translate(50 52) scale(0.82) translate(-50 -52)"/>
        case "devoted":
            return <>
                <path d={FLAME} fill={accent}/>
                <path d={FLAME_INNER} fill={ink}/>
            </>
        case "unbroken":
            return <>
                <circle cx="50" cy="51" r="27" stroke={ink} strokeWidth="3" strokeDasharray="5 4" fill="none"/>
                <path d={FLAME} fill={accent} transform={SMALLER}/>
                <path d={FLAME_INNER} fill={ink} transform={SMALLER}/>
            </>
        case "talker":
            return <>
                <path d={BUBBLE} fill={accent}/>
                {[40, 50, 60].map((x) => <circle key={x} cx={x} cy="43" r="3.6" fill={ink}/>)}
            </>
        case "chatterbox":
            return <>
                <path d="M44 26 Q44 22 48 22 H68 Q72 22 72 26 V40 Q72 44 68 44 H66 V50 L60 44 H48 Q44 44 44 40 Z"
                      stroke={ink} strokeWidth="3.5" strokeLinejoin="round" fill="none"/>
                <path d="M26 40 Q26 36 30 36 H54 Q58 36 58 40 V58 Q58 62 54 62 H40 L31 71 V62 H30 Q26 62 26 58 Z" fill={accent}/>
                {[34, 42, 50].map((x) => <circle key={x} cx={x} cy="49" r="3" fill={ink}/>)}
            </>
        case "socialite":
            return <>
                <path d={BUBBLE} fill={accent}/>
                <path d="M50 53 C40 46 37 41 40 36.5 C43 32 48.5 33 50 37.5 C51.5 33 57 32 60 36.5 C63 41 60 46 50 53 Z" fill={ink}/>
            </>
        case "icon":
            return <>
                <path d={BUBBLE} fill={accent}/>
                <path d="M50 32 L52.7 39.3 L60.5 39.6 L54.4 44.4 L56.5 51.9 L50 47.6 L43.5 51.9 L45.6 44.4 L39.5 39.6 L47.3 39.3 Z" fill={ink}/>
            </>
        default:
            return <text x="50" y="63" textAnchor="middle" fontFamily="Luckiest Guy, sans-serif" fontSize="34"
                         fill={accent}>{title.name.slice(0, 1)}</text>
    }
}

export type TitleEmblemProps = {
    title: PlayerTitle
    size: number
    locked?: boolean
    ribbon?: boolean
}

export default function TitleEmblem({title, size, locked = false, ribbon = false}: TitleEmblemProps) {
    const id = `metal-${useId().replace(/:/g, "")}`
    const stops = METALS[metalOf(title)]
    const viewHeight = ribbon ? 110 : 100

    if (locked) {
        return <svg className="title-emblem title-emblem-locked" width={size} height={size} viewBox="0 0 100 100" aria-hidden="true">
            <circle cx="50" cy="50" r="46" fill="#262626"/>
            {TICKS.map((tick, i) => <circle key={i} cx={tick.x} cy={tick.y} r="1.6" fill="#00000033"/>)}
            <circle cx="50" cy="50" r="36" fill="#141414" stroke="#000000" strokeWidth="2"/>
            {icon(title, "#474747", "#3A3A3A")}
            <circle cx="76" cy="76" r="13" fill="#101010" stroke="#3A3A3A" strokeWidth="2"/>
            <rect x="70" y="75" width="12" height="9" rx="2" fill="#8A8A8A"/>
            <path d="M72.5 75 V72 A3.5 3.5 0 0 1 79.5 72 V75" stroke="#8A8A8A" strokeWidth="2.4" fill="none"/>
        </svg>
    }

    return <svg className="title-emblem" width={size} height={size * viewHeight / 100} viewBox={`0 0 100 ${viewHeight}`}
                style={{filter: `drop-shadow(0 0 10px ${stops[1]}66) drop-shadow(0 4px 0 #000000AA)`}}
                aria-hidden="true">
        <defs>
            <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
                {stops.map((stop, i) => <stop key={stop} offset={`${i * 100 / (stops.length - 1)}%`} stopColor={stop}/>)}
            </linearGradient>
        </defs>
        {ribbon && <>
            <path d="M30 80 L22 100 L32 95 L37 104 L44 84 Z" fill={`url(#${id})`} opacity="0.9"/>
            <path d="M70 80 L78 100 L68 95 L63 104 L56 84 Z" fill={`url(#${id})`} opacity="0.9"/>
        </>}
        <circle cx="50" cy="50" r="46" fill={`url(#${id})`}/>
        {TICKS.map((tick, i) => <circle key={i} cx={tick.x} cy={tick.y} r="1.6" fill="#00000055"/>)}
        <circle cx="50" cy="50" r="36" fill={enamelOf(title)} stroke="#00000066" strokeWidth="2"/>
        {icon(title, "#FFF7E6", `url(#${id})`)}
        <path d="M20 38 A32 32 0 0 1 46 16" stroke="#FFFFFF" strokeOpacity="0.55" strokeWidth="4" strokeLinecap="round" fill="none"/>
    </svg>
}
