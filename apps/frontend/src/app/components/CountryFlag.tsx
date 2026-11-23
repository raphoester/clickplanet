import {CSSProperties, useEffect} from "react";
import {BLANK_REGION, Region, regions} from "../viewer/atlas.ts";
import {ATLAS_SIZE, ATLAS_URL} from "../viewer/atlasAsset.ts";
import {TITLE_CAP_HEIGHT} from "../titleFont.ts";
import {knowsFlag, useFlags} from "../flags/flagsStore.ts";
import type {FlagPicture} from "../../domain/flags.ts";
import "./CountryFlag.css"

const BOX = {width: 1, height: TITLE_CAP_HEIGHT} as const
const BOX_STYLE: CSSProperties = {width: `${BOX.width}em`, height: `${BOX.height}em`}

const sprites = new Map<string, CSSProperties>()
const BLANK_STYLE = frameOf(BLANK_REGION)

export type CountryFlagProps = {
    code: string,
}

// A built flag is its sprite in the atlas, a sold one its own picture; a flag with neither, one the page does not know
// yet, is a plain grey flag.
export default function CountryFlag(props: CountryFlagProps) {
    const built = regions.has(props.code)
    const picture = useFlags((store) => built ? undefined : store.catalog.pictureOf(props.code))
    const pictured = built || picture !== undefined
    useEffect(() => {
        if (!pictured && props.code !== "") knowsFlag(props.code)
    }, [props.code, pictured])
    if (props.code === "") return null

    return <span className="country-flag" aria-hidden="true" style={BOX_STYLE}>
        <span className={pictured ? "country-flag-sprite" : "country-flag-sprite country-flag-blank"}
              style={picture ? soldStyle(picture) : spriteStyle(props.code)}/>
    </span>
}

function soldStyle(picture: FlagPicture): CSSProperties {
    return {
        ...frameOf({x: 0, y: 0, width: picture.width, height: picture.height}),
        backgroundImage: `url("${encodeURI(picture.url)}")`,
        backgroundSize: "100% 100%",
    }
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
