import * as THREE from "three"
import {latticeSize} from "../../domain/lattice.ts"
import {decodeTerrain, type Land, LAND} from "../../domain/terrain.ts"

// Must match TERRAIN_COLUMNS in shaders/territory/fragment.glsl.
const TERRAIN_COLUMNS = 1024

// The board-game look the owner chose in November 2026.
const LAND_COLOURS: Record<Land, string> = {
    forest: "#2e8b47",
    tundra: "#a9b79a",
    boreal: "#3d7d55",
    arid: "#c98a4b",
    savanna: "#b6b84e",
    grass: "#6caf43",
    sand: "#e8c77a",
    snow: "#e9eff3",
}

// One lattice step from land, two, three, then the open sea.
const SEA_COLOURS = ["#4a9bd6", "#3a86c6", "#2e72b3", "#245f9e"]

// The shaders write a colour as it is, so the hex goes in with no conversion, or it comes out dark.
const raw = (hex: string) => new THREE.Color().setStyle(hex, THREE.LinearSRGBColorSpace)

export const OPEN_SEA_COLOUR = raw(SEA_COLOURS[SEA_COLOURS.length - 1])

// Every kind's colour, in the terrain's order.
export const terrainColours = (): THREE.Color[] => [...LAND.map((kind) => LAND_COLOURS[kind]), ...SEA_COLOURS].map(raw)

export async function loadTerrain(url: string, cols: number, signal?: AbortSignal): Promise<Uint8Array> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return decodeTerrain(await response.arrayBuffer(), latticeSize(cols))
}

// One texel a lattice point, read by canonical id.
export function terrainTexture(terrain: Uint8Array): THREE.DataTexture {
    const rows = Math.ceil(terrain.length / TERRAIN_COLUMNS)
    const data = new Uint8Array(TERRAIN_COLUMNS * rows)
    data.set(terrain)
    const texture = new THREE.DataTexture(data, TERRAIN_COLUMNS, rows, THREE.RedIntegerFormat, THREE.UnsignedByteType)
    texture.internalFormat = "R8UI"
    texture.needsUpdate = true
    return texture
}
