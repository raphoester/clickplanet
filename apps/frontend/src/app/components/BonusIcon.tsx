import {BonusReward} from '../../domain/bonus.ts'

export default function BonusIcon({kind}: { kind: BonusReward["kind"] }) {
    return <svg className="bonus-icon" viewBox="0 0 48 48" aria-hidden="true">
        {DRAWINGS[kind]}
    </svg>
}

function hexagon(cx: number, cy: number, r: number): string {
    return Array.from({length: 6}, (_, i) => {
        const angle = Math.PI / 6 + i * Math.PI / 3
        return `${(cx + r * Math.cos(angle)).toFixed(2)},${(cy + r * Math.sin(angle)).toFixed(2)}`
    }).join(" ")
}

function around(cx: number, cy: number, distance: number, count: number): [number, number][] {
    return Array.from({length: count}, (_, i) => {
        const angle = i * 2 * Math.PI / count
        return [cx + distance * Math.cos(angle), cy + distance * Math.sin(angle)]
    })
}

const DRAWINGS: Record<BonusReward["kind"], React.ReactNode> = {
    refill: <>
        <path className="bonus-icon-line" d="M7 14 H37 V34 H7 Z"/>
        <path className="bonus-icon-ink" d="M37 20 H42 V28 H37 Z"/>
        <path className="bonus-icon-ink" d="M11 18 H17 V30 H11 Z M19 18 H25 V30 H19 Z M27 18 H33 V30 H27 Z"/>
    </>,

    spreadClicks: <>
        {around(24, 24, 15.5, 6).map(([x, y]) =>
            <polygon key={`${x},${y}`} className="bonus-icon-ink bonus-icon-ink--soft"
                     points={hexagon(x, y, 6.5)}/>)}
        <polygon className="bonus-icon-ink" points={hexagon(24, 24, 9)}/>
    </>,

    bomb: <>
        <path className="bonus-icon-fuse" d="M32 14 Q34 5 40 8"/>
        <rect className="bonus-icon-dark" x="27" y="11" width="9" height="8" rx="1.5"
              transform="rotate(40 31.5 15)"/>
        <circle className="bonus-icon-dark" cx="22" cy="29" r="14"/>
        <path className="bonus-icon-shine" d="M13 27 A 9 9 0 0 1 20 20"/>
        <polygon className="bonus-icon-spark"
                 points="41,2 42.6,6.4 47,8 42.6,9.6 41,14 39.4,9.6 35,8 39.4,6.4"/>
    </>,

    encloseClicks: <>
        <polygon className="bonus-icon-ink bonus-icon-ink--soft" points={hexagon(24, 24, 11)}/>
        {around(24, 24, 17.5, 12).map(([x, y]) =>
            <polygon key={`${x},${y}`} className="bonus-icon-ink" points={hexagon(x, y, 3.9)}/>)}
        <polygon className="bonus-icon-spark"
                 points="24,18 25.5,22.5 30,24 25.5,25.5 24,30 22.5,25.5 18,24 22.5,22.5"/>
    </>,

    defenders: <>
        <path className="bonus-icon-ink" d="M24 5 L39 10 V22 C39 32 32 39 24 43 C16 39 9 32 9 22 V10 Z"/>
        <path className="bonus-icon-ink bonus-icon-ink--soft" d="M24 11 L33 14 V22 C33 28 29 33 24 36 Z"/>
        <polygon className="bonus-icon-spark"
                 points="24,16 26,21.5 31.5,23.5 26,25.5 24,31 22,25.5 16.5,23.5 22,21.5"/>
    </>,
}
