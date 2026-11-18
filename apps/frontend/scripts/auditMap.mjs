import fs from "node:fs"
import path from "node:path"

import {latticeIds} from "../src/domain/lattice.ts"
import {LAND} from "../src/domain/terrain.ts"
import {boardAsked, boardDir, readBoard, readBorders, readBridges, readCapitals, readProvinces, readTerrain, staticDir, tileOfPoint} from "./map/board.mjs"
import {bridgesOf, lanesOf, piecesApart, snapKmOf} from "./map/bridges.mjs"
import {SEA, loadGround} from "./map/ground.mjs"
import {lattice, lonLatOf, neighbours} from "./map/lattice.mjs"
import {NATURAL_EARTH_TAG} from "./map/naturalEarth.mjs"
import {latticeGround} from "./map/oracle.mjs"
import {cutOf} from "./map/provinces.mjs"
import {REGIONS, seatsOf} from "./map/seats.mjs"
import {terrainOf} from "./map/terrain.mjs"

const ANTIMERIDIAN_DEGREES = 1

async function main() {
    // `npm run map:audit -- earth-2`; earth-1 when no board is named.
    const definition = boardAsked(process.argv.slice(2))
    const tiles = readBoard(boardDir(definition.name))
    const borders = readBorders(tiles)
    const ground = await loadGround()
    const {lattice: shape} = tiles.manifest
    const {count: vertices, positions} = lattice(shape)
    const edges = neighbours(shape)
    const {grounds, lakes, seated} = latticeGround(ground, {count: vertices, positions}, edges, seatsOf(tiles.manifest.name))

    const tileOf = tileOfPoint(tiles, vertices)

    const counts = {
        vertices,
        tiles: tiles.count,
        groundIsLand: 0,
        antarcticIce: 0,
    }
    const faults = {
        tilesOnSea: 0,
        landWithNoTile: 0,
        iceWithNoTile: 0,
        tilesWithNoCountry: 0,
        flagsWithNoGround: 0,
        groundWithNoCapital: 0,
        capitalsOffTheirGround: 0,
        bridgesAstray: 0,
        landNoBridgeReaches: 0,
        terrainAstray: 0,
        ...(definition.provinceFloor ? {tilesInNoProvince: 0, provincesOffTheirLandmass: 0, provincesUnderTheFloor: 0, provincesAstray: 0} : {}),
    }
    const seam = {land: 0, tiles: 0}
    const elsewhere = {land: 0, tiles: 0}

    for (let i = 0; i < vertices; i++) {
        const tile = tileOf[i] >= 0
        const [lon, lat] = lonLatOf(positions, i)
        const land = grounds[i] !== SEA

        if (land) counts.groundIsLand++
        if (tile && !land) faults.tilesOnSea++
        if (land && !tile) faults.landWithNoTile++

        const near = Math.abs(lon) > 180 - ANTIMERIDIAN_DEGREES
        const band = near ? seam : elsewhere
        if (land) band.land++
        if (tile) band.tiles++
    }

    const ice = await iceOnly()
    for (let i = 0; i < vertices; i++) {
        const [lon, lat] = lonLatOf(positions, i)
        if (lat > -55) continue
        if (ice.groundOf(lon, lat) === SEA) continue
        counts.antarcticIce++
        if (tileOf[i] < 0) faults.iceWithNoTile++
    }

    for (let t = 0; t < borders.count; t++) if (borders.landmass[t] === 0) faults.tilesWithNoCountry++

    const flags = JSON.parse(fs.readFileSync(path.join(staticDir, "countries", "countries.json"), "utf8"))
    const grounded = new Set(borders.codes.slice(1))
    const groundless = Object.keys(flags).filter((code) => !REGIONS.has(code) && !grounded.has(code))
    faults.flagsWithNoGround = groundless.length

    const {capitals} = readCapitals(tiles)
    for (const code of grounded) if (!(code in capitals)) faults.groundWithNoCapital++
    for (const [code, tile] of Object.entries(capitals)) {
        if (borders.codes[borders.landmass[tile - 1]] !== code) faults.capitalsOffTheirGround++
    }

    const crossings = {at: edges.at, to: edges.to, tileOf}
    const drawn = bridgesOf({positions: tiles.positions, ...crossings, capitals}, lanesOf(tiles.manifest.name), snapKmOf(tiles.manifest.spacing)).bridges
    const {bridges} = readBridges(tiles)
    const kept = new Set(bridges.map((pair) => pair.join(",")))
    faults.bridgesAstray = drawn.filter((pair) => !kept.has(pair.join(","))).length
        + Math.max(0, bridges.length - drawn.length)
    faults.landNoBridgeReaches = piecesApart(crossings, tiles.count, bridges)

    // The terrain rebuilt from its own kinds of land on this board's tiles: a land kind off a tile, a tile at sea or a
    // sea band at the wrong distance from land all differ.
    const {terrain} = readTerrain(tiles)
    const kinds = Uint8Array.from(tiles.ids, (id) => terrain[id] < LAND.length ? terrain[id] : 0)
    const rebuilt = terrainOf(kinds, crossings, latticeIds(shape))
    for (let id = 0; id < terrain.length; id++) if (rebuilt[id] !== terrain[id]) faults.terrainAstray++

    const provinces = definition.provinceFloor ? await auditProvinces(tiles, borders, definition.provinceFloor, faults) : undefined

    report({counts, faults, seam, elsewhere, tiles, borders, seated, groundless, lakes, provinces})

    const broken = Object.entries(faults).filter(([, n]) => n > 0)
    if (broken.length === 0) {
        console.log("\nthe tile set and the oracle agree about every lattice vertex")
        return
    }

    console.error("\nthe tile set and the oracle disagree, which they cannot do if the board came"
        + " from one `npm run map:generate`:")
    for (const [name, n] of broken) console.error(`  ${name}: ${n}`)
    process.exitCode = 1
}

// Every tile in a province on its own landmass, every province of the floor or more but a whole landmass or a whole UK
// nation, and the file as a fresh cut writes it.
async function auditProvinces(tiles, borders, floor, faults) {
    let file
    try {
        file = readProvinces(tiles)
    } catch (error) {
        console.error(String(error))
        faults.tilesInNoProvince++
        return undefined
    }
    const {province, landmass, tiles: sizes} = file.provinces
    for (let t = 0; t < tiles.count; t++) if (landmass[province[t]] !== borders.landmass[t]) faults.provincesOffTheirLandmass++

    const {cut} = await cutOf(tiles.manifest.name, floor)
    for (let t = 0; t < tiles.count; t++) if (cut.province[t] !== province[t]) faults.provincesAstray++

    const onLandmass = new Map()
    const ofNation = new Map()
    for (let k = 1; k < landmass.length; k++) {
        onLandmass.set(landmass[k], (onLandmass.get(landmass[k]) ?? 0) + 1)
        const nation = cut.nations[k] ?? ""
        if (nation) ofNation.set(`${landmass[k]} ${nation}`, (ofNation.get(`${landmass[k]} ${nation}`) ?? 0) + 1)
    }
    const small = {wholeLandmasses: 0, wholeNations: 0}
    for (let k = 1; k < landmass.length; k++) {
        if (sizes[k] >= floor) continue
        if (onLandmass.get(landmass[k]) === 1) small.wholeLandmasses++
        else if (cut.nations[k] && ofNation.get(`${landmass[k]} ${cut.nations[k]}`) === 1) small.wholeNations++
        else faults.provincesUnderTheFloor++
    }
    return {name: file.name, count: landmass.length - 1, floor, ...small}
}

async function iceOnly() {
    const {groundIndex} = await import("./map/ground.mjs")
    const {naturalEarth} = await import("./map/naturalEarth.mjs")
    return groundIndex({
        countries: {features: []},
        iceShelves: await naturalEarth("ne_50m_antarctic_ice_shelves_polys"),
    })
}

function report({counts, faults, seam, elsewhere, tiles, borders, seated, groundless, lakes, provinces}) {
    const pct = (n, of) => `${(n / of * 100).toFixed(2)}%`
    console.log(`Natural Earth ${NATURAL_EARTH_TAG} | ${tiles.manifest.name} ${tiles.file} | ${borders.name}`)
    console.log(`
lattice vertices        ${counts.vertices}
  tiles                 ${counts.tiles}  (${pct(counts.tiles, counts.vertices)})
  ground says land      ${counts.groundIsLand}
  antarctic ice shelf   ${counts.antarcticIce}

faults between the tile set and the ground oracle
  tiles on open sea     ${faults.tilesOnSea}
  land with no tile     ${faults.landWithNoTile}
  ice shelf with no tile ${faults.iceWithNoTile}
  tiles in no country   ${faults.tilesWithNoCountry}
  flags with no ground  ${faults.flagsWithNoGround}${groundless.length ? `  (${groundless.join(" ")})` : ""}
  ground with no capital ${faults.groundWithNoCapital}
  capitals off their ground ${faults.capitalsOffTheirGround}
  bridges astray from the generator ${faults.bridgesAstray}
  land no bridge reaches ${faults.landNoBridgeReaches}
  terrain astray from the tiles ${faults.terrainAstray}${provinces === undefined ? "" : `
  tiles in no province  ${faults.tilesInNoProvince}
  provinces off their landmass ${faults.provincesOffTheirLandmass}
  provinces under ${provinces.floor} tiles ${faults.provincesUnderTheFloor}, but whole landmasses and whole nations
  provinces astray from the generator ${faults.provincesAstray}`}

lakes, sea on a country's ground where most of the cell is lake
  ${lakes} lattice vertices

seats, one tile each for a country the lattice misses
  ${seated.length} seated, ${seated.filter(({was}) => was === SEA).length} of them at sea

the antimeridian, within ${ANTIMERIDIAN_DEGREES} degree of the dateline
  land ${seam.land}, tiles ${seam.tiles}  -> ${seam.land ? (seam.tiles / seam.land).toFixed(3) : "n/a"} tiles per land vertex
  elsewhere: land ${elsewhere.land}, tiles ${elsewhere.tiles} -> ${(elsewhere.tiles / elsewhere.land).toFixed(3)}`)
    if (provinces !== undefined) {
        console.log(`
provinces, ${provinces.name}
  ${provinces.count} provinces; under ${provinces.floor} tiles, ${provinces.wholeLandmasses} whole landmasses and ${provinces.wholeNations} whole UK nations`)
    }
}

await main()
