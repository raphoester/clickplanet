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
const SHIELD = "M50 27 L69 33 V48 C69 61 61 69 50 74 C39 69 31 61 31 48 V33 Z"
const SHIELD_INNER = "M50 37 L60 40 V48 C60 55 56 60 50 63 C44 60 40 55 40 48 V40 Z"
const TOWER = "M37 73 V42 H35 V31 H41 V36 H47 V31 H53 V36 H59 V31 H65 V42 H63 V73 Z"
const CASTLE = "M30 73 V38 H34 V42 H38 V38 H42 V52 H58 V38 H62 V42 H66 V38 H70 V73 Z"
const GATE = "M44 73 V63 C44 57 56 57 56 63 V73 Z"
const BOMB = "M33 57 A14 14 0 1 0 61 57 A14 14 0 1 0 33 57 Z"
const BOMB_CAP = "M53 40 L60 47 L64 43 L57 36 Z"
const FUSE = "M60 39 Q63 31 70 29"
const SPARK = "M70.0 22.0 L71.8 26.6 L76.7 26.8 L72.9 29.9 L74.1 34.7 L70.0 32.0 L65.9 34.7 L67.1 29.9 L63.3 26.8 L68.2 26.6 Z"
const BLAST = "M50.0 26.0 L54.6 38.9 L67.0 33.0 L61.1 45.4 L74.0 50.0 L61.1 54.6 L67.0 67.0 L54.6 61.1 L50.0 74.0 L45.4 61.1 L33.0 67.0 L38.9 54.6 L26.0 50.0 L38.9 45.4 L33.0 33.0 L45.4 38.9 Z"
const BLAST_INNER = "M50.0 38.0 L52.3 44.5 L58.5 41.5 L55.5 47.7 L62.0 50.0 L55.5 52.3 L58.5 58.5 L52.3 55.5 L50.0 62.0 L47.7 55.5 L41.5 58.5 L44.5 52.3 L38.0 50.0 L44.5 47.7 L41.5 41.5 L47.7 44.5 Z"
const BOX = "M32 38 H68 V70 H32 Z"
const SACK = "M43 33 H57 L53 41 C64 45 69 54 68 62 C67 70 59 74 50 74 C41 74 33 70 32 62 C31 54 36 45 47 41 Z"
const CHEST = "M30 50 H70 V71 H30 Z"
const CHEST_LID = "M30 50 V43 C30 33 70 33 70 43 V50 Z"
const COMPASS = "M30 50 A20 20 0 1 0 70 50 A20 20 0 1 0 30 50"
const GLOBE = "M31 50 A19 19 0 1 0 69 50 A19 19 0 1 0 31 50 Z"
const MAP = "M29 37 L42 32 L58 38 L71 33 V63 L58 68 L42 62 L29 67 Z"
const SCOPE = "M31 50 A19 19 0 1 0 69 50 A19 19 0 1 0 31 50"
const PUMPKIN = "M50 41 C40 37 28 44 28 56 C28 68 39 73 50 71 C61 73 72 68 72 56 C72 44 60 37 50 41 Z"
const PUMPKIN_FACE = "M38 55 L45 55 L41.5 48 Z M55 55 L62 55 L58.5 48 Z M37 60 L43 64.5 L47 61.5 L50 66 L53 61.5 L57 64.5 L63 60 Q50 72 37 60 Z"
const STEM = "var(--halloween)"
const GLOW = "var(--pumpkin-glow)"
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

function Bomb({accent, cream}: {accent: string, cream: string}) {
    return <>
        <Line d={FUSE} color={cream} width={3}/>
        <Shape d={BOMB_CAP} color={cream} width={2.5}/>
        <Shape d={BOMB} color={accent}/>
        <path d="M39 52 C40 49 42 47 45 46" style={{stroke: LIGHT}} strokeOpacity="0.85" strokeWidth="3"
              strokeLinecap="round" fill="none"/>
    </>
}

function Target({accent, cream, x = 50, y = 50, r = 20}: {accent: string, cream: string, x?: number, y?: number, r?: number}) {
    return <>
        <circle cx={x} cy={y} r={r} style={{fill: accent, stroke: INK}} strokeWidth="3"/>
        <circle cx={x} cy={y} r={r * 0.6} style={{fill: cream, stroke: INK}} strokeWidth="3"/>
    </>
}

function Meridians() {
    return <path d="M50 31 C41 37 41 63 50 69 M50 31 C59 37 59 63 50 69 M31 50 H69" style={{stroke: INK}}
                 strokeWidth="2.5" fill="none"/>
}

function Letters({text, color}: {text: string, color: string}) {
    return <text x="50" y="63" textAnchor="middle" fontSize="32" strokeWidth="5" strokeLinejoin="round"
                 style={{fill: color, stroke: INK, paintOrder: "stroke", fontFamily: "var(--font-display)"}}>{text}</text>
}

const LAUREL = "M45 71 C35 67 29 58 31 35"
const LAUREL_LEAVES = [[36, 66, -60], [30, 58, -35], [28, 48, -10], [29, 38, 15]]

function Laurel({place, accent, cream}: {place: string, accent: string, cream: string}) {
    return <>
        {["", "translate(100 0) scale(-1 1)"].map((mirror) => <g key={mirror} transform={mirror || undefined}>
            <Line d={LAUREL} color={cream} width={2.5}/>
            {LAUREL_LEAVES.map(([x, y, turn]) => <ellipse key={y} cx={x} cy={y} rx="3.4" ry="6.2"
                                                         transform={`rotate(${turn} ${x} ${y})`}
                                                         style={{fill: cream, stroke: INK}} strokeWidth="2"/>)}
        </g>)}
        <Letters text={place} color={accent}/>
    </>
}

function icon(title: PlayerTitle, accent: string, cream: string): ReactNode {
    switch (title.id) {
        case "og":
            return <Letters text="OG" color={accent}/>
        case "season_0_gold":
            return <Letters text="1" color={accent}/>
        case "season_0_silver":
            return <Letters text="2" color={accent}/>
        case "season_0_bronze":
            return <Letters text="3" color={accent}/>
        case "season_0_top_1":
            return <Laurel place="1" accent={accent} cream={cream}/>
        case "season_0_top_2":
            return <Laurel place="2" accent={accent} cream={cream}/>
        case "season_0_top_3":
            return <Laurel place="3" accent={accent} cream={cream}/>
        case "season_0_veteran":
            return <Shape d={STAR} color={accent} width={1.6} transform="translate(50 50) scale(2) translate(-50 -43)"/>
        case "halloween_2026":
            return <>
                <Shape d="M48 42 L47 32 Q51 28 56 31 L53 42 Z" color={STEM} width={2.5}/>
                <Shape d={PUMPKIN} color={accent}/>
                <path d="M44 42 Q37 56 44 70 M56 42 Q63 56 56 70" style={{stroke: "var(--pumpkin-shade)"}} strokeWidth="2.5"
                      fill="none"/>
                <Shape d={PUMPKIN_FACE} color={GLOW} width={2}/>
            </>
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
        case "guard":
            return <Shape d={SHIELD} color={accent}/>
        case "defender":
            return <>
                <Shape d={SHIELD} color={accent}/>
                <Shape d={SHIELD_INNER} color={cream} width={2.5}/>
            </>
        case "protector":
            return <>
                <Shape d={TOWER} color={accent}/>
                <Shape d={GATE} color={cream} width={2.5}/>
            </>
        case "warden":
            return <>
                <Line d="M50 52 V24" color={cream} width={3}/>
                <Shape d="M52 24 L65 29 L52 34 Z" color={cream} width={2.5}/>
                <Shape d={CASTLE} color={accent}/>
                <Shape d={GATE} color={cream} width={2.5}/>
            </>
        case "bomber":
            return <Bomb accent={accent} cream={cream}/>
        case "blaster":
            return <>
                <Bomb accent={accent} cream={cream}/>
                <Shape d={SPARK} color={accent} width={2.5}/>
            </>
        case "wrecker":
            return <>
                <Shape d={BLAST} color={accent}/>
                <circle cx="50" cy="50" r="6" style={{fill: cream, stroke: INK}} strokeWidth="2.5"/>
            </>
        case "destroyer":
            return <>
                <circle cx="50" cy="50" r="29" style={{stroke: cream}} strokeWidth="3" strokeDasharray="5 6" fill="none"/>
                <Shape d={BLAST} color={accent}/>
                <Shape d={BLAST_INNER} color={cream} width={2.5}/>
            </>
        case "looter":
            return <>
                <Shape d={BOX} color={accent}/>
                <Letters text="?" color={cream}/>
            </>
        case "hunter":
            return <>
                <Shape d={SACK} color={accent}/>
                <Line d="M44 41 H56" color={cream} width={3}/>
            </>
        case "pirate":
            return <>
                <Shape d={CHEST_LID} color={accent}/>
                <Shape d={CHEST} color={accent}/>
                <Shape d="M45 46 H55 V57 H45 Z" color={cream} width={2.5}/>
            </>
        case "captain":
            return <>
                <Shape d={CHEST_LID} color={accent}/>
                <Shape d={CHEST} color={accent}/>
                <Shape d={STAR} color={cream} width={2.5} transform="translate(0 14)"/>
            </>
        case "student":
            return <>
                <Line d={COMPASS} color={cream} width={3.5}/>
                <Shape d="M50 32 L56 50 L44 50 Z" color={accent} width={2.5}/>
                <Shape d="M50 68 L56 50 L44 50 Z" color={cream} width={2.5}/>
            </>
        case "expert":
            return <>
                <Shape d={GLOBE} color={accent}/>
                <Meridians/>
            </>
        case "professor":
            return <>
                <Shape d={MAP} color={accent}/>
                <path d="M42 32 V62 M58 38 V68" style={{stroke: INK}} strokeWidth="2.5" fill="none"/>
                <Line d="M47 44 L53 50 M53 44 L47 50" color={cream} width={3}/>
            </>
        case "genius":
            return <g transform="translate(50 46) scale(0.85) translate(-50 -50)">
                <Line d="M30 34 A26 26 0 0 0 68 72" color={cream} width={3.5}/>
                <Shape d={GLOBE} color={accent}/>
                <Meridians/>
                <Line d="M50 76 V84 M40 86 H60" color={cream} width={3.5}/>
            </g>
        case "rookie":
            return <>
                <Target accent={accent} cream={cream} r={23}/>
                <circle cx="50" cy="50" r="6" style={{fill: accent, stroke: INK}} strokeWidth="2.5"/>
            </>
        case "agent":
            return <>
                <Line d={SCOPE} color={accent} width={4}/>
                <Line d="M50 25 V39 M50 61 V75 M25 50 H39 M61 50 H75" color={cream} width={3}/>
                <circle cx="50" cy="50" r="4.5" style={{fill: accent, stroke: INK}} strokeWidth="2.5"/>
            </>
        case "operative":
            return <>
                <Target accent={accent} cream={cream} x={47} y={53} r={19}/>
                <Line d="M47 53 L70 30" color={cream} width={3}/>
                <Line d="M61 39 V32 M61 39 H68 M66 34 V27 M66 34 H73" color={accent} width={3}/>
            </>
        case "spymaster":
            return <>
                <circle cx="50" cy="50" r="29" style={{stroke: cream}} strokeWidth="3" strokeDasharray="5 6" fill="none"/>
                <Target accent={accent} cream={cream} r={21}/>
                <Shape d={STAR} color={accent} width={2.5} transform="translate(0 8)"/>
            </>
        default:
            return <Letters text={title.name.slice(0, 1)} color={accent}/>
    }
}

function Bands({colors}: {colors: readonly string[]}) {
    const length = BAND_LENGTH / colors.length
    return <>
        {colors.map((color, i) => <circle key={i} cx="50" cy="50" r={BAND_RADIUS} fill="none"
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
