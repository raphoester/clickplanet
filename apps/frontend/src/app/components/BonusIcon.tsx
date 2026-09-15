import {BonusReward} from '../../domain/bonus.ts'

/**
 * The drawing in the bonus box, one per kind.
 *
 * Drawn rather than an emoji: an emoji is a different picture on every platform
 * and never matches the box it sits in. These take their outline from
 * `--bonus-box-edge`, so each one is inked in its own box's colour. The styles
 * are in BonusAward.css.
 */
export default function BonusIcon({kind}: { kind: BonusReward["kind"] }) {
    return <svg className="bonus-icon" viewBox="0 0 48 48" aria-hidden="true">
        {DRAWINGS[kind]}
    </svg>
}

/** A pointy-top hexagon, the shape of a tile on the planet. */
function hexagon(cx: number, cy: number, r: number): string {
    return Array.from({length: 6}, (_, i) => {
        const angle = Math.PI / 6 + i * Math.PI / 3
        return `${(cx + r * Math.cos(angle)).toFixed(2)},${(cy + r * Math.sin(angle)).toFixed(2)}`
    }).join(" ")
}

/** `count` points evenly spaced on a circle around a centre. */
function around(cx: number, cy: number, distance: number, count: number): [number, number][] {
    return Array.from({length: count}, (_, i) => {
        const angle = i * 2 * Math.PI / count
        return [cx + distance * Math.cos(angle), cy + distance * Math.sin(angle)]
    })
}

const DRAWINGS: Record<BonusReward["kind"], React.ReactNode> = {
    // A pointer clicking, with three sparks at its tip: three clicks for one.
    tripleClicks: <>
        <path className="bonus-icon-line" d="M20 12 L20 4 M15 14 L9.5 8.5 M13 19 L5 19"/>
        <path className="bonus-icon-ink"
              d="M20 19 L20 43 L26.5 37 L31 46 L36 43.5 L31.5 35 L40 34 Z"/>
    </>,

    // One tile taking the six around it.
    spreadClicks: <>
        {around(24, 24, 15.5, 6).map(([x, y]) =>
            <polygon key={`${x},${y}`} className="bonus-icon-ink bonus-icon-ink--soft"
                     points={hexagon(x, y, 6.5)}/>)}
        <polygon className="bonus-icon-ink" points={hexagon(24, 24, 9)}/>
    </>,

    // A round bomb with a lit fuse.
    bomb: <>
        <path className="bonus-icon-fuse" d="M32 14 Q34 5 40 8"/>
        <rect className="bonus-icon-dark" x="27" y="11" width="9" height="8" rx="1.5"
              transform="rotate(40 31.5 15)"/>
        <circle className="bonus-icon-dark" cx="22" cy="29" r="14"/>
        <path className="bonus-icon-shine" d="M13 27 A 9 9 0 0 1 20 20"/>
        <polygon className="bonus-icon-spark"
                 points="41,2 42.6,6.4 47,8 42.6,9.6 41,14 39.4,9.6 35,8 39.4,6.4"/>
    </>,

    // A loop of tiles closed, and the ground inside it taken.
    encloseClicks: <>
        <polygon className="bonus-icon-ink bonus-icon-ink--soft" points={hexagon(24, 24, 11)}/>
        {around(24, 24, 17.5, 12).map(([x, y]) =>
            <polygon key={`${x},${y}`} className="bonus-icon-ink" points={hexagon(x, y, 3.9)}/>)}
        <polygon className="bonus-icon-spark"
                 points="24,18 25.5,22.5 30,24 25.5,25.5 24,30 22.5,25.5 18,24 22.5,22.5"/>
    </>,
}
