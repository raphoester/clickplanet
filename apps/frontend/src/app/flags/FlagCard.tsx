import {ReactNode, useId} from "react"
import {t} from "../../i18n/page.ts"
import {CrownIcon} from "../components/icons.tsx"
import "./FlagSubmission.css"

export type FlagCardProps = {
    name: string
    // A picture the player picked, or the public one once approved; none drawn as bare ground.
    picture?: string
    // The capital's place, or its coordinates.
    capital?: string
    chip?: ReactNode
}

// A tile's radius, about a tile's size on the globe at the first zoom.
const R = 13
const W = Math.sqrt(3) * R
const ROWS = [4, 5, 4]
const DRAWN = R * 0.9
const HEX = [-90, -30, 30, 90, 150, 210]
    .map((degrees) => degrees * Math.PI / 180)
    .map((angle, i) => `${i === 0 ? "M" : "L"}${(DRAWN * Math.cos(angle)).toFixed(2)} ${(DRAWN * Math.sin(angle)).toFixed(2)}`)
    .join(" ") + "Z"
const WIDTH = Math.max(...ROWS) * W
const HEIGHT = 2 * R + (ROWS.length - 1) * 1.5 * R
const TILES = ROWS.flatMap((count, row) => Array.from({length: count}, (_, i) => ({
    x: (Math.max(...ROWS) - count) * W / 2 + W / 2 + i * W,
    y: R + row * 1.5 * R,
})))

// The flag as the players will see it: a few tiles wearing its picture around its capital, its name and the capital's
// place.
export default function FlagCard({name, picture, capital, chip}: FlagCardProps) {
    const clip = `flag-card-${useId().replace(/:/g, "")}`

    return <div className="panel-box flag-card">
        <span className="flag-card-tiles" style={{width: WIDTH, height: HEIGHT}} aria-hidden="true">
            <svg viewBox={`0 0 ${WIDTH.toFixed(2)} ${HEIGHT}`} width={WIDTH} height={HEIGHT}>
                <defs>
                    <clipPath id={clip}><path d={HEX}/></clipPath>
                </defs>
                {TILES.map(({x, y}) => <g key={`${x} ${y}`} transform={`translate(${x.toFixed(2)} ${y})`}>
                    {picture
                        ? <image href={picture}
                                 x={-W / 2}
                                 y={-R}
                                 width={W}
                                 height={2 * R}
                                 preserveAspectRatio="xMidYMid slice"
                                 clipPath={`url(#${clip})`}/>
                        : <path d={HEX} className="flag-card-ground"/>}
                </g>)}
            </svg>
            <span className="flag-card-crown"><CrownIcon size={16}/></span>
        </span>
        <span className="flag-card-text">
            <span className={name ? "flag-card-name" : "flag-card-name flag-card-name--empty"}>
                {name || t("flagSubmission.namePlaceholder")}
            </span>
            {capital && <span className="flag-card-capital"><CrownIcon size={14}/>{capital}</span>}
            {chip}
        </span>
    </div>
}
