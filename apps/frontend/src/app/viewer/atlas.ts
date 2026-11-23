import atlasRegion from "../../../static/countries/atlas.json"

export type Region = {
    x: number,
    y: number,
    width: number,
    height: number,
}

export const regions: Map<string, Region> = new Map(Object.entries(atlasRegion))

// Each flag's number on the GPU, from 1: 0 is no flag.
export const flagNumbers: Map<string, number> = new Map(Array.from(regions.keys(), (code, i) => [code, i + 1]))

const codes = [...regions.keys()]

export function flagCodeOf(number: number): string | undefined {
    return codes[number - 1]
}
