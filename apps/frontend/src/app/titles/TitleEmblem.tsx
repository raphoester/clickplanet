import {ReactNode, useId} from "react"
import {PlayerTitle} from "../../backends/player.ts"
import {BANDS, enamelOf, METALS, metalOf, ribbonOf} from "./titleArt.ts"

const INK = "var(--ink)"
const LIGHT = "var(--text)"
const CREAM = "var(--cream)"
const LOCKED = {
    ring: "color-mix(in srgb, var(--text-faint) 35%, var(--panel))",
    accent: "color-mix(in srgb, var(--text-faint) 50%, var(--panel))",
    cream: "color-mix(in srgb, var(--text-faint) 70%, var(--panel))",
    enamel: "var(--panel-sunk)",
}

const FLAME = "M50 26 C59 39 67 46 67 59 C67 70 59 76 50 76 C41 76 33 70 33 59 C33 50 39 45 41 37 C45 44 47 47 50 50 C51 41 48 34 50 26 Z"
const FLAME_INNER = "M50 50 C55 56 59 60 59 65 C59 71 55 74 50 74 C45 74 41 71 41 65 C41 60 45 57 50 50 Z"
const BUBBLE = "M28 32 Q28 26 34 26 H66 Q72 26 72 32 V54 Q72 60 66 60 H48 L37 71 V60 H34 Q28 60 28 54 Z"
const HEART = "M50 53 C40 46 37 41 40 36.5 C43 32 48.5 33 50 37.5 C51.5 33 57 32 60 36.5 C63 41 60 46 50 53 Z"
const STAR = "M50 32 L52.7 39.3 L60.5 39.6 L54.4 44.4 L56.5 51.9 L50 47.6 L43.5 51.9 L45.6 44.4 L39.5 39.6 L47.3 39.3 Z"
const SMALLER = "translate(50 49) scale(0.9) translate(-50 -49)"
const LOYAL = "translate(50 52) scale(0.82) translate(-50 -52)"
const UNBROKEN = "translate(50 52) scale(0.7) translate(-50 -52)"
const RIBBONS = ["M32 74 L20 104 L32 98 L38 109 L49 80 Z", "M68 74 L80 104 L68 98 L62 109 L51 80 Z"]
const BAND_RADIUS = 39.5
const BAND_LENGTH = 2 * Math.PI * BAND_RADIUS

const RIVETS = Array.from({length: 12}, (_, i) => {
    const angle = i * Math.PI / 6 + Math.PI / 12
    return {x: 50 + 40 * Math.cos(angle), y: 50 + 40 * Math.sin(angle)}
})

function Line({d, color, width = 4}: {d: string, color: string, width?: number}) {
    return <>
        <path d={d} style={{stroke: INK}} strokeWidth={width + 4} strokeLinecap="round" strokeLinejoin="round" fill="none"/>
        <path d={d} style={{stroke: color}} strokeWidth={width} strokeLinecap="round" strokeLinejoin="round" fill="none"/>
    </>
}

function Shape({d, color, width = 3, transform}: {d: string, color: string, width?: number, transform?: string}) {
    return <path d={d} style={{fill: color, stroke: INK}} strokeWidth={width} strokeLinejoin="round" transform={transform}/>
}

function Dots({xs, y, r}: {xs: number[], y: number, r: number}) {
    return <>{xs.map((x) => <circle key={x} cx={x} cy={y} r={r} style={{fill: INK}}/>)}</>
}

function Letters({text, color}: {text: string, color: string}) {
    return <text x="50" y="63" textAnchor="middle" fontSize="32" strokeWidth="5" strokeLinejoin="round"
                 style={{fill: color, stroke: INK, paintOrder: "stroke", fontFamily: "var(--font-display)"}}>{text}</text>
}

function icon(title: PlayerTitle, accent: string, cream: string): ReactNode {
    switch (title.id) {
        case "og":
            return <Letters text="OG" color={accent}/>
        case "settler":
            return <>
                <Line d="M42 71 V30" color={cream}/>
                <Shape d="M44 30 L68 37 L44 45 Z" color={accent}/>
                <Line d="M31 71 H61" color={cream}/>
            </>
        case "raider":
            return <>
                <Line d="M36 70 L59 38" color={cream}/>
                <Shape d="M56 28 C68 28 74 38 72 48 L58 40 L52 44 Z" color={accent}/>
            </>
        case "warlord":
            return <>
                <Shape d="M37 48 C27 45 22 36 25 25 C29 33 34 37 41 40 Z" color={cream}/>
                <Shape d="M63 48 C73 45 78 36 75 25 C71 33 66 37 59 40 Z" color={cream}/>
                <Shape d="M32 62 C32 44 40 36 50 36 C60 36 68 44 68 62 L58 62 L58 54 L42 54 L42 62 Z" color={accent}/>
                <Line d="M50 55 V66" color={cream} width={3}/>
                <path d="M39 46 C41 42 44 40 48 39.5" style={{stroke: LIGHT}} strokeOpacity="0.85" strokeWidth="3"
                      strokeLinecap="round" fill="none"/>
            </>
        case "conqueror":
            return <>
                <Line d="M34 66 L66 34" color={cream}/>
                <Line d="M66 66 L34 34" color={cream}/>
                <Line d="M35 55 L45 65" color={accent}/>
                <Line d="M65 55 L55 65" color={accent}/>
            </>
        case "warmaster":
            return <>
                <Line d="M44 76 V31" color={cream}/>
                <Shape d="M44 19 L50 30 L38 30 Z" color={cream}/>
                <Shape d="M46 32 H72 L64 42 L72 52 H46 Z" color={accent}/>
                <path d="M55 37 L56.8 40.8 L61 41.2 L57.8 44 L58.8 48 L55 45.8 L51.2 48 L52.2 44 L49 41.2 L53.2 40.8 Z"
                      style={{fill: INK}}/>
                <Line d="M35 76 H53" color={cream}/>
            </>
        case "loyal":
            return <Shape d={FLAME} color={accent} width={3.6} transform={LOYAL}/>
        case "devoted":
            return <>
                <Shape d={FLAME} color={accent}/>
                <Shape d={FLAME_INNER} color={cream} width={2.5}/>
            </>
        case "unbroken":
            return <>
                <circle cx="50" cy="51" r="27" style={{stroke: cream}} strokeWidth="3.5" strokeDasharray="6 5" fill="none"/>
                <Shape d={FLAME} color={accent} width={4} transform={UNBROKEN}/>
                <Shape d={FLAME_INNER} color={cream} width={3.5} transform={UNBROKEN}/>
            </>
        case "talker":
            return <g transform={SMALLER}>
                <Shape d={BUBBLE} color={accent}/>
                <Dots xs={[40, 50, 60]} y={43} r={3.6}/>
            </g>
        case "chatterbox":
            return <g transform={SMALLER}>
                <Line d="M44 26 Q44 22 48 22 H68 Q72 22 72 26 V40 Q72 44 68 44 H66 V50 L60 44 H48 Q44 44 44 40 Z"
                      color={cream} width={3}/>
                <Shape d="M26 40 Q26 36 30 36 H54 Q58 36 58 40 V58 Q58 62 54 62 H40 L31 71 V62 H30 Q26 62 26 58 Z" color={accent}/>
                <Dots xs={[34, 42, 50]} y={49} r={3}/>
            </g>
        case "socialite":
            return <g transform={SMALLER}>
                <Shape d={BUBBLE} color={accent}/>
                <Shape d={HEART} color={cream} width={2.5}/>
            </g>
        case "icon":
            return <g transform={SMALLER}>
                <Shape d={BUBBLE} color={accent}/>
                <Shape d={STAR} color={cream} width={2.5}/>
            </g>
        default:
            return <Letters text={title.name.slice(0, 1)} color={accent}/>
    }
}

function Bands({colors}: {colors: readonly string[]}) {
    const length = BAND_LENGTH / colors.length
    return <>
        {colors.map((color, i) => <circle key={color} cx="50" cy="50" r={BAND_RADIUS} fill="none"
                                          style={{stroke: color}} strokeWidth="11"
                                          strokeDasharray={`${length} ${BAND_LENGTH - length}`}
                                          strokeDashoffset={-i * length}
                                          transform="rotate(-90 50 50)"/>)}
    </>
}

function Padlock() {
    const shackle = "M74 79 V75.5 A5 5 0 0 1 84 75.5 V79"
    return <>
        <circle cx="79" cy="80" r="14" style={{fill: "var(--panel)", stroke: INK}} strokeWidth="3"/>
        <path d={shackle} style={{stroke: INK}} strokeWidth="5" fill="none"/>
        <path d={shackle} style={{stroke: "var(--text-soft)"}} strokeWidth="2" fill="none"/>
        <rect x="71.5" y="78" width="15" height="11" rx="2.5" style={{fill: "var(--text-soft)", stroke: INK}} strokeWidth="2.5"/>
    </>
}

export type TitleEmblemProps = {
    title: PlayerTitle
    size: number
    locked?: boolean
    ribbon?: boolean
}

export default function TitleEmblem({title, size, locked = false, ribbon = false}: TitleEmblemProps) {
    const id = `medal-${useId().replace(/:/g, "")}`
    const metal = metalOf(title)
    const ring = locked ? LOCKED.ring : METALS[metal]
    const accent = locked ? LOCKED.accent : METALS[metal]
    const bands = locked ? undefined : BANDS[metal]
    const viewHeight = ribbon || locked ? 114 : 102

    return <svg className={locked ? "title-emblem title-emblem-locked" : "title-emblem"}
                width={size}
                height={size * viewHeight / 100}
                viewBox={`0 0 100 ${viewHeight}`}
                aria-hidden="true">
        <defs>
            <mask id={`${id}-shade`}>
                <circle cx="50" cy="50" r="45" style={{fill: LIGHT}}/>
                <circle cx="45" cy="44" r="45" style={{fill: INK}}/>
            </mask>
            <mask id={`${id}-light`}>
                <circle cx="50" cy="50" r="45" style={{fill: LIGHT}}/>
                <circle cx="53" cy="54" r="45" style={{fill: INK}}/>
            </mask>
        </defs>
        {ribbon && !locked && <>
            {RIBBONS.map((d) => <path key={`drop ${d}`} d={d} style={{fill: INK}} transform="translate(0 4)"/>)}
            {RIBBONS.map((d) => <path key={d} d={d} style={{fill: ribbonOf(title), stroke: INK}} strokeWidth="3" strokeLinejoin="round"/>)}
        </>}
        <circle cx="50" cy="55" r="46" style={{fill: INK}}/>
        <circle cx="50" cy="50" r="45" style={{fill: ring}}/>
        {bands && <Bands colors={bands}/>}
        <circle cx="50" cy="50" r="45" style={{fill: INK}} fillOpacity="0.28" mask={`url(#${id}-shade)`}/>
        <circle cx="50" cy="50" r="45" style={{fill: LIGHT}} fillOpacity="0.45" mask={`url(#${id}-light)`}/>
        {RIVETS.map((rivet, i) => <circle key={i} cx={rivet.x} cy={rivet.y} r="1.8" style={{fill: INK}} fillOpacity="0.3"/>)}
        <circle cx="50" cy="50" r="45" fill="none" style={{stroke: INK}} strokeWidth="4"/>
        <circle cx="50" cy="50" r="33" style={{fill: locked ? LOCKED.enamel : enamelOf(title), stroke: INK}} strokeWidth="3.5"/>
        {icon(title, accent, locked ? LOCKED.cream : CREAM)}
        {locked
            ? <Padlock/>
            : <path d="M14 37 A39 39 0 0 1 28 18" style={{stroke: LIGHT}} strokeWidth="4.5" strokeLinecap="round" fill="none"/>}
    </svg>
}
