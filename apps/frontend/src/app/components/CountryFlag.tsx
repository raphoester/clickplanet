import {CSSProperties} from "react";
import {regions} from "../viewer/atlas.ts";
import {ATLAS_SIZE, ATLAS_URL} from "../viewer/atlasAsset.ts";
import {TITLE_CAP_HEIGHT} from "../titleFont.ts";
import "./CountryFlag.css"

// The box a flag is drawn in, in em, whatever its own aspect ratio: flags are as
// wide as they fit and centred, so a column of them lines up on both edges.
//
// The height is Luckiest Guy's cap height, and the box sits on the baseline — so
// it covers exactly the band the capitals next to it cover, at 20px in the
// leaderboard as at 24px in the folded header. A flag sized in px cannot do
// that: it drifts low as the text around it grows. The number is in
// `titleFont.ts`, which says why anything beside a title needs it.
const BOX = {width: 1, height: TITLE_CAP_HEIGHT} as const
const BOX_STYLE: CSSProperties = {width: `${BOX.width}em`, height: `${BOX.height}em`}

// The leaderboard re-renders all 255 rows on every batch of updates, and the
// picker lists as many again. The cut is the same every time, so hold onto it:
// React skips the style diff entirely when the object is the one it already saw.
const sprites = new Map<string, CSSProperties>()

export type CountryFlagProps = {
    code: string,
}

export default function CountryFlag(props: CountryFlagProps) {
    const sprite = spriteStyle(props.code)
    if (!sprite) return null

    return <span className="country-flag" aria-hidden="true" style={BOX_STYLE}>
        <span className="country-flag-sprite" style={sprite}/>
    </span>
}

function spriteStyle(code: string): CSSProperties | undefined {
    const known = sprites.get(code)
    if (known) return known

    const region = regions.get(code)
    if (!region) return undefined

    // em per pixel of atlas, so the whole cut scales with the text it stands in.
    const scale = Math.min(BOX.width / region.width, BOX.height / region.height)
    const height = region.height * scale

    const style: CSSProperties = {
        width: `${region.width * scale}em`,
        height: `${height}em`,
        // Centred by hand rather than by the box: a flex box takes its baseline
        // from the item inside it, which would hang a short flag lower than a
        // tall one and leave a column of them jittering.
        marginTop: `${(BOX.height - height) / 2}em`,
        backgroundImage: `url("${ATLAS_URL}")`,
        backgroundSize: `${ATLAS_SIZE.width * scale}em ${ATLAS_SIZE.height * scale}em`,
        backgroundPosition: `${-region.x * scale}em ${-region.y * scale}em`,
    }

    sprites.set(code, style)
    return style
}
