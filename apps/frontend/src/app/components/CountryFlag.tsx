import {CSSProperties, useEffect} from "react";
import {BLANK_REGION, Region, regions} from "../viewer/atlas.ts";
import {ATLAS_SIZE, ATLAS_URL} from "../viewer/atlasAsset.ts";
import {TITLE_CAP_HEIGHT} from "../titleFont.ts";
import {knowsFlag} from "../flags/flagsStore.ts";
import "./CountryFlag.css"

const BOX = {width: 1, height: TITLE_CAP_HEIGHT} as const
const BOX_STYLE: CSSProperties = {width: `${BOX.width}em`, height: `${BOX.height}em`}

const sprites = new Map<string, CSSProperties>()
const BLANK_STYLE = frameOf(BLANK_REGION)

export type CountryFlagProps = {
    code: string,
}

// A flag with no picture in the atlas, one opened since the build, is a plain grey flag.
export default function CountryFlag(props: CountryFlagProps) {
    const pictured = regions.has(props.code)
    useEffect(() => {
        if (!pictured && props.code !== "") knowsFlag(props.code)
    }, [props.code, pictured])
    if (props.code === "") return null

    return <span className="country-flag" aria-hidden="true" style={BOX_STYLE}>
        <span className={pictured ? "country-flag-sprite" : "country-flag-sprite country-flag-blank"}
              style={spriteStyle(props.code)}/>
    </span>
}

function spriteStyle(code: string): CSSProperties {
    const known = sprites.get(code)
    if (known) return known

    const region = regions.get(code)
    const style = region ? {...frameOf(region), ...pictureOf(region)} : BLANK_STYLE
    if (region) sprites.set(code, style)
    return style
}

function frameOf(region: Region): CSSProperties {
    const scale = scaleOf(region)
    const height = region.height * scale
    return {width: `${region.width * scale}em`, height: `${height}em`, marginTop: `${(BOX.height - height) / 2}em`}
}

function pictureOf(region: Region): CSSProperties {
    const scale = scaleOf(region)
    return {
        backgroundImage: `url("${ATLAS_URL}")`,
        backgroundSize: `${ATLAS_SIZE.width * scale}em ${ATLAS_SIZE.height * scale}em`,
        backgroundPosition: `${-region.x * scale}em ${-region.y * scale}em`,
    }
}

function scaleOf(region: Region): number {
    return Math.min(BOX.width / region.width, BOX.height / region.height)
}
