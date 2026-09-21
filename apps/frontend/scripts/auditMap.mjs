// Asks the three sources that used to answer "sea or land" whether they still disagree.
//
//   npm run map:audit          report, and fail if any fault got worse than the baseline
//   npm run map:audit -- --save   rewrite the baseline from this run
//
// The three are the shipped tile set (which lattice vertices became tiles), the ground oracle
// (Natural Earth, which also says which country), and the globe's own texture (what a player
// actually sees). The texture is not an authority and never becomes one — a 4096x2048 photo blends
// a one-tile island into open water — so it is reported rather than enforced. The faults that are
// enforced are the ones between the tile set and the oracle.
//
// It is run by hand: it downloads 3 MB, rebuilds the whole 906,012-vertex lattice and needs a
// raised heap. `npm run map:audit` sets that up.
import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

import sharp from "sharp"

import {readBorders, readCoordinates, staticDir} from "./map/blob.mjs"
import {SEA, loadGround} from "./map/ground.mjs"
import {keyOf, lattice, lonLatOf} from "./map/lattice.mjs"
import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"

const baselinePath = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "map", "audit.baseline.json")
const save = process.argv.includes("--save")

// Blue clearly dominant. Calibrated against the oracle rather than guessed: it agrees with the
// country polygons on 99.1% of the lattice, and what it misses is what a photo misses — islands
// smaller than a few pixels, and ice shelves that read as land whoever you ask.
const seaPixel = (r, g, b) => b > r + 18 && b > g + 8

const ANTIMERIDIAN_DEGREES = 1

async function main() {
    const tiles = readCoordinates()
    const borders = readBorders()
    const ground = await loadGround()
    const {count: vertices, positions, uvs} = lattice()

    const texture = await sharp(path.join(staticDir, "earth", "earth-4k.jpg"))
        .raw().toBuffer({resolveWithObject: true})
    const {width, height, channels} = texture.info
    const textureSaysLand = (u, v) => {
        const x = Math.min(Math.floor(u * width), width - 1)
        const y = Math.min(Math.max(height - 1 - Math.floor(v * height), 0), height - 1)
        const i = (y * width + x) * channels
        return !seaPixel(texture.data[i], texture.data[i + 1], texture.data[i + 2])
    }

    const isTile = new Set()
    for (let t = 0; t < tiles.count; t++) {
        isTile.add(keyOf(tiles.positions[t * 3], tiles.positions[t * 3 + 1], tiles.positions[t * 3 + 2]))
    }

    const counts = {
        vertices,
        tiles: tiles.count,
        groundIsLand: 0,
        textureIsLand: 0,
        antarcticIce: 0,
    }
    const faults = {
        tilesOnSea: 0,
        landWithNoTile: 0,
        iceWithNoTile: 0,
        tilesWithNoCountry: 0,
    }
    const texture_ = {landButBlue: 0, blueButLand: 0}
    const seam = {land: 0, tiles: 0}
    const elsewhere = {land: 0, tiles: 0}

    for (let i = 0; i < vertices; i++) {
        const tile = isTile.has(keyOf(positions[i * 3], positions[i * 3 + 1], positions[i * 3 + 2]))
        const [lon, lat] = lonLatOf(uvs[i * 2], uvs[i * 2 + 1])
        const code = ground.groundOf(lon, lat)
        const land = code !== SEA
        const drawnAsLand = textureSaysLand(uvs[i * 2], uvs[i * 2 + 1])

        if (land) counts.groundIsLand++
        if (drawnAsLand) counts.textureIsLand++
        if (tile && !land) faults.tilesOnSea++
        if (land && !tile) faults.landWithNoTile++
        if (land && !drawnAsLand) texture_.landButBlue++
        if (!land && drawnAsLand) texture_.blueButLand++

        const near = Math.abs(lon) > 180 - ANTIMERIDIAN_DEGREES
        const band = near ? seam : elsewhere
        if (land) band.land++
        if (tile) band.tiles++
    }

    // The ice shelves on their own, so "Antarctica has holes in it" is a number rather than a
    // guess. They are in no country polygon, so before this they were sea to every generator.
    const ice = await iceOnly()
    for (let i = 0; i < vertices; i++) {
        const [lon, lat] = lonLatOf(uvs[i * 2], uvs[i * 2 + 1])
        if (lat > -55) continue
        if (ice.groundOf(lon, lat) === SEA) continue
        counts.antarcticIce++
        if (!isTile.has(keyOf(positions[i * 3], positions[i * 3 + 1], positions[i * 3 + 2]))) faults.iceWithNoTile++
    }

    for (let t = 0; t < borders.count; t++) if (borders.landmass[t] === 0) faults.tilesWithNoCountry++

    report({counts, faults, texture: texture_, seam, elsewhere, tiles, borders})

    const baseline = fs.existsSync(baselinePath) ? JSON.parse(fs.readFileSync(baselinePath, "utf8")) : null
    if (save) {
        fs.writeFileSync(baselinePath, `${JSON.stringify({naturalEarth: NATURAL_EARTH_TAG, faults}, null, 4)}\n`)
        console.log(`\nwrote ${path.relative(process.cwd(), baselinePath)}`)
        return
    }
    if (!baseline) {
        console.log("\nno baseline recorded yet — run with --save")
        return
    }

    const worse = Object.entries(faults).filter(([name, n]) => n > (baseline.faults[name] ?? 0))
    if (worse.length === 0) {
        console.log("\nno fault is worse than the baseline")
        return
    }
    for (const [name, n] of worse) console.error(`${name}: ${n}, baseline ${baseline.faults[name] ?? 0}`)
    process.exitCode = 1
}

// A second oracle holding the shelves alone. Building it twice costs a second and keeps `groundOf`
// answering one thing; a "which layer answered" channel on the oracle would only exist for this.
async function iceOnly() {
    const {groundIndex} = await import("./map/ground.mjs")
    const {naturalEarth} = await import("./map/naturalEarth.mjs")
    return groundIndex({
        countries: {features: []},
        iceShelves: await naturalEarth("ne_50m_antarctic_ice_shelves_polys"),
    })
}

function report({counts, faults, texture, seam, elsewhere, tiles, borders}) {
    const pct = (n, of) => `${(n / of * 100).toFixed(2)}%`
    console.log(`Natural Earth ${NATURAL_EARTH_TAG} | ${tiles.name} | ${borders.name}`)
    console.log(`
lattice vertices        ${counts.vertices}
  tiles                 ${counts.tiles}  (${pct(counts.tiles, counts.vertices)})
  ground says land      ${counts.groundIsLand}
  texture looks like land ${counts.textureIsLand}
  antarctic ice shelf   ${counts.antarcticIce}

faults between the tile set and the ground oracle
  tiles on open sea     ${faults.tilesOnSea}
  land with no tile     ${faults.landWithNoTile}
  ice shelf with no tile ${faults.iceWithNoTile}
  tiles in no country   ${faults.tilesWithNoCountry}

the texture, which follows rather than decides
  land drawn as water   ${texture.landButBlue}
  water drawn as land   ${texture.blueButLand}

the antimeridian, within ${ANTIMERIDIAN_DEGREES} degree of the dateline
  land ${seam.land}, tiles ${seam.tiles}  -> ${seam.land ? (seam.tiles / seam.land).toFixed(3) : "n/a"} tiles per land vertex
  elsewhere: land ${elsewhere.land}, tiles ${elsewhere.tiles} -> ${(elsewhere.tiles / elsewhere.land).toFixed(3)}`)
}

await main()
