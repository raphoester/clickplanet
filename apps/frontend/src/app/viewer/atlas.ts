import atlasRegion from "../../../static/countries/atlas.json"
import {ATLAS_SIZE} from "./atlasAsset.ts"

export type Region = {
    x: number,
    y: number,
    width: number,
    height: number,
}

export const regions: Map<string, Region> = new Map(Object.entries(atlasRegion))

// Each flag's number on the GPU, from 1: 0 is no flag.
export const flagNumbers: Map<string, number> = new Map(Array.from(regions.keys(), (code, i) => [code, i + 1]))

const BLANK_SIZE = {width: 24, height: 16}
// A sample at a lower mip level reaches past a rectangle's edge.
const BLANK_MARGIN = 4

// A flag with no picture in the atlas, one opened since the build, wears a blank of it: the shaders draw it white.
export const BLANK_REGION: Region = blankOf(regions.values(), ATLAS_SIZE)
export const BLANK_NUMBER = regions.size + 1
// Every number a flag takes on the GPU, the blank's included.
export const FLAG_NUMBERS = BLANK_NUMBER + 1

export function flagNumberOf(code: string): number {
    return flagNumbers.get(code) ?? BLANK_NUMBER
}

export function regionOf(code: string): Region {
    return regions.get(code) ?? BLANK_REGION
}

// A rectangle below or beside a picture that comes near none: the atlas is transparent where no picture is.
export function blankOf(pictures: Iterable<Region>, size: {width: number, height: number}): Region {
    const all = [...pictures]
    const apart = (a: Region, b: Region) => a.x + a.width + BLANK_MARGIN <= b.x || b.x + b.width + BLANK_MARGIN <= a.x
        || a.y + a.height + BLANK_MARGIN <= b.y || b.y + b.height + BLANK_MARGIN <= a.y
    const inside = (r: Region) => r.x + r.width <= size.width && r.y + r.height <= size.height
    for (const picture of all) {
        const below = {x: picture.x + BLANK_MARGIN, y: picture.y + picture.height + BLANK_MARGIN, ...BLANK_SIZE}
        const beside = {x: picture.x + picture.width + BLANK_MARGIN, y: picture.y + BLANK_MARGIN, ...BLANK_SIZE}
        const blank = [below, beside].find((r) => inside(r) && all.every((other) => apart(r, other)))
        if (blank) return blank
    }
    throw new Error("the flag atlas has no room for a blank flag: pack it with some")
}
