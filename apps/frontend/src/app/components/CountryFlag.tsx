import {CSSProperties} from "react";
import {regions} from "../viewer/atlas.ts";
import {ATLAS_SIZE, ATLAS_URL} from "../viewer/atlasAsset.ts";
import {TITLE_CAP_HEIGHT} from "../titleFont.ts";
import "./CountryFlag.css"

const BOX = {width: 1, height: TITLE_CAP_HEIGHT} as const
const BOX_STYLE: CSSProperties = {width: `${BOX.width}em`, height: `${BOX.height}em`}

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

    const scale = Math.min(BOX.width / region.width, BOX.height / region.height)
    const height = region.height * scale

    const style: CSSProperties = {
        width: `${region.width * scale}em`,
        height: `${height}em`,
        marginTop: `${(BOX.height - height) / 2}em`,
        backgroundImage: `url("${ATLAS_URL}")`,
        backgroundSize: `${ATLAS_SIZE.width * scale}em ${ATLAS_SIZE.height * scale}em`,
        backgroundPosition: `${-region.x * scale}em ${-region.y * scale}em`,
    }

    sprites.set(code, style)
    return style
}
