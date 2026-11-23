import atlasRegion from "../../../static/countries/atlas.json"
import {ATLAS_SIZE} from "./atlasAsset.ts"
import type {FlagPicture} from "../../domain/flags.ts"

export type Region = {
    x: number,
    y: number,
    width: number,
    height: number,
}

export const regions: Map<string, Region> = new Map(Object.entries(atlasRegion))

// A built sprite's width, and a sold flag's in the atlas.
export const SPRITE_WIDTH = 100

// Each flag's number on the GPU, from 1: 0 is no flag.
export const flagNumbers: Map<string, number> = new Map(Array.from(regions.keys(), (code, i) => [code, i + 1]))

const BLANK_SIZE = {width: 24, height: 16}
// A sample at a lower mip level reaches past a rectangle's edge.
const BLANK_MARGIN = 4

// A flag whose picture is not on the GPU yet wears a blank of the atlas: the shaders draw it white.
export const BLANK_REGION: Region = blankOf(regions.values(), ATLAS_SIZE)
export const BLANK_NUMBER = regions.size + 1
// The flags opened since the build, numbered after the blank as the page first meets them: each wears its own picture,
// drawn under the built atlas, in the same texture (flagAtlas.ts). A flag past them wears the blank.
export const FIRST_SOLD = BLANK_NUMBER + 1
export const SOLD_MOST = 256
// Every number a flag takes on the GPU, the blank's and the sold ones' included.
export const FLAG_NUMBERS = FIRST_SOLD + SOLD_MOST

const soldNumbers = new Map<string, number>()
const soldCodes: string[] = []
// The sold flags whose picture is in the globe's texture: its cell there.
const shown = new Map<number, Region>()
const pictures = new Map<string, FlagPicture>()

export function flagNumberOf(code: string): number {
    const built = flagNumbers.get(code)
    if (built !== undefined) return built
    const sold = soldNumbers.get(code)
    if (sold !== undefined) return sold
    if (code === "" || soldCodes.length >= SOLD_MOST) return BLANK_NUMBER
    const number = FIRST_SOLD + soldCodes.length
    soldCodes.push(code)
    soldNumbers.set(code, number)
    return number
}

export function regionOf(code: string): Region {
    return regions.get(code) ?? regionOfNumber(flagNumberOf(code))
}

export function regionOfNumber(number: number): Region {
    return shown.get(number) ?? builtRegions[number - 1] ?? BLANK_REGION
}

// The globe's texture holds the flag's picture in its cell, or no longer does.
export function showPicture(number: number, cell: Region | undefined): void {
    if (cell === undefined) shown.delete(number)
    else shown.set(number, cell)
}

export function knowPicture(code: string, picture: FlagPicture): void {
    pictures.set(code, picture)
}

export function pictureOf(code: string): FlagPicture | undefined {
    return pictures.get(code)
}

// Width over height: the built sprite's, or the sold flag's picture; none for a flag with neither.
export function aspectOf(code: string): number | undefined {
    const region = regions.get(code)
    if (region) return region.width / region.height
    const picture = pictures.get(code)
    return picture ? picture.width / picture.height : undefined
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

const codes = [...regions.keys()]
const builtRegions = [...regions.values()]

export function flagCodeOf(number: number): string | undefined {
    return codes[number - 1] ?? soldCodes[number - FIRST_SOLD]
}
