import {mkdirSync, writeFileSync} from "node:fs"
import {join, resolve} from "node:path"

import {PNG} from "pngjs"

import {cellsOf, latticeFaces, tileAt} from "../src/domain/latticeCells.ts"
import {BOARD_DEFINITIONS} from "./map/board.mjs"
import {positionOf} from "./map/seats.mjs"
import {cutOf} from "./map/provinces.mjs"

// `npm run map:provinces -- earth-2 100,200,400 [--out <dir>]`: a board's provinces at each floor, as PNGs of a few
// crops, one color a province and the country borders in black. Nothing is written to the board.
const CROPS = {
    world: {west: -180, east: 180, south: -60, north: 84, width: 2400},
    europe: {west: -12, east: 42, south: 34, north: 71, width: 1600},
    usa: {west: -126, east: -66, south: 24, north: 50, width: 1600},
}
const SEA = [214, 228, 240]
const BORDER = [0, 0, 0]

const args = process.argv.slice(2)
const out = resolve(args.includes("--out") ? args[args.indexOf("--out") + 1] : join("node_modules", ".cache", "provinces"))
const [board, floors] = args.filter((arg, i) => !arg.startsWith("-") && args[i - 1] !== "--out")
if (BOARD_DEFINITIONS[board] === undefined || floors === undefined) {
    throw new Error("usage: npm run map:provinces -- <board> <N,N,...> [--out <dir>]")
}
mkdirSync(out, {recursive: true})

for (const floor of floors.split(",").map(Number)) {
    const {board: tiles, borders, cut} = await cutOf(board, floor)
    const cells = cellsOf(tiles.ids, tiles.manifest.lattice)
    const faces = latticeFaces(tiles.manifest.lattice)
    const count = cut.totals.length - 1
    let ofFloor = 0
    for (let k = 1; k <= count; k++) if (cut.totals[k] >= floor) ofFloor++

    for (const [crop, {west, east, south, north, width}] of Object.entries(CROPS)) {
        const across = Math.cos(((south + north) / 2) * Math.PI / 180)
        const height = Math.round(width * (north - south) / ((east - west) * (crop === "world" ? 1 : across)))
        const under = new Int32Array(width * height)
        for (let row = 0; row < height; row++) {
            const lat = north - (row + 0.5) / height * (north - south)
            for (let column = 0; column < width; column++) {
                const lon = west + (column + 0.5) / width * (east - west)
                const [x, y, z] = positionOf([lon, lat])
                under[row * width + column] = (tileAt(cells, faces, x, y, z) ?? 0) - 1
            }
        }

        const country = (pixel: number) => under[pixel] < 0 ? "" : borders.codes[borders.landmass[under[pixel]]]
        const png = new PNG({width, height})
        for (let pixel = 0; pixel < width * height; pixel++) {
            const column = pixel % width
            const around = [
                column > 0 ? pixel - 1 : pixel, column < width - 1 ? pixel + 1 : pixel,
                pixel >= width ? pixel - width : pixel, pixel + width < width * height ? pixel + width : pixel,
            ]
            const edge = around.some((other) => country(other) !== country(pixel))
            const color = edge ? BORDER : under[pixel] < 0 ? SEA : colorOf(cut.province[under[pixel]])
            png.data.set([...color, 255], pixel * 4)
        }
        const file = join(out, `provinces-${board}-${floor}-${crop}.png`)
        writeFileSync(file, PNG.sync.write(png))
        console.log(`${file}: ${count} provinces, ${ofFloor} of ${floor} tiles or more`)
    }
}

// A light color of its own for each province, the same at every run.
function colorOf(province: number): number[] {
    let hash = Math.imul(province ^ 0x9e3779b9, 0x85ebca6b)
    hash = Math.imul(hash ^ (hash >>> 13), 0xc2b2ae35)
    hash ^= hash >>> 16
    const hue = (hash >>> 0) % 360
    const [s, l] = [0.55 + ((hash >>> 9) & 15) / 60, 0.62 + ((hash >>> 13) & 15) / 80]
    const a = s * Math.min(l, 1 - l)
    const f = (n: number) => {
        const k = (n + hue / 30) % 12
        return Math.round(255 * (l - a * Math.max(-1, Math.min(k - 3, 9 - k, 1))))
    }
    return [f(0), f(8), f(4)]
}
