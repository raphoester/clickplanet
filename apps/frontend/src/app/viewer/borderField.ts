import * as THREE from "three"
import type {OwnerChange} from "../../domain/tileOwnership.ts"
import {regions} from "./atlas.ts"
import flagFit from "../../../static/countries/flagFit.json"

const LANDMASS_TEXELS = 4

export type BorderData = {
    codes: string[]
    assignment: Uint16Array
    frames: Float32Array
    totals: Uint32Array
}

export async function loadBorders(url: string, signal?: AbortSignal): Promise<BorderData> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const buffer = await response.arrayBuffer()
    const headerBytes = new DataView(buffer).getUint32(0, true)
    const header = JSON.parse(new TextDecoder().decode(new Uint8Array(buffer, 4, headerBytes)))
    let at = 4 + headerBytes

    const assignment = new Uint16Array(buffer, at, header.tiles)
    // A Float32Array view must start on 4 bytes; the encoder pads an odd tile count to match.
    at =Math.ceil((at + header.tiles * 2) / 4) * 4
    const frames = new Float32Array(buffer, at, header.codes.length * 5)
    at += header.codes.length * 5 * 4
    const totals = new Uint32Array(buffer, at, header.codes.length)

    return {codes: header.codes, assignment, frames, totals}
}

export function countryOfTile(data: BorderData, tile: number): string | undefined {
    const index = data.assignment[tile - 1]
    return index ? data.codes[index] : undefined
}

export class BorderField {
    readonly landmassData: THREE.DataTexture
    readonly landmassCount: number

    private readonly held: Map<string, number>[]
    private readonly ownerOf: (string | undefined)[]
    private readonly rows: Float32Array
    private readonly painted: (string | undefined)[]

    constructor(
        private readonly data: BorderData,
        tileCount: number,
        private readonly minimumShare = MINIMUM_SHARE,
        private readonly contrast = CONTRAST,
        private readonly minimumTiles = MINIMUM_TILES,
        private readonly stretch = true,
        private readonly fit: "cover" | "contain" = "cover",
    ) {
        this.landmassCount = data.codes.length
        this.ownerOf = new Array(tileCount + 1)
        this.held = Array.from({length: this.landmassCount}, () => new Map<string, number>())
        this.painted = new Array(this.landmassCount)

        this.rows = new Float32Array(LANDMASS_TEXELS * this.landmassCount * 4)
        for (let piece = 1; piece < this.landmassCount; piece++) {
            const at = piece * LANDMASS_TEXELS * 4
            this.rows[at] = data.frames[piece * 5]
            this.rows[at + 1] = data.frames[piece * 5 + 1]
            this.rows[at + 2] = data.frames[piece * 5 + 2]
            this.rows[at + 4] = 0
            this.rows[at + 5] = 0
            this.rows[at + 6] = 0
        }

        this.landmassData = new THREE.DataTexture(
            this.rows, LANDMASS_TEXELS, this.landmassCount, THREE.RGBAFormat, THREE.FloatType,
        )
        this.landmassData.minFilter = THREE.NearestFilter
        this.landmassData.magFilter = THREE.NearestFilter
        this.landmassData.needsUpdate = true
    }

    holderOf(landmass: number): string | undefined {
        return this.painted[landmass]
    }

    apply(changes: OwnerChange[]) {
        const touched = new Set<number>()

        for (const {tile, country} of changes) {
            const index = this.data.assignment[tile - 1]
            if (!index) continue

            const previous = this.ownerOf[tile]
            if (previous === country) continue

            const held = this.held[index]
            if (previous !== undefined) {
                const left = (held.get(previous) ?? 1) - 1
                if (left > 0) held.set(previous, left)
                else held.delete(previous)
            }
            if (country !== undefined) held.set(country, (held.get(country) ?? 0) + 1)

            this.ownerOf[tile] = country
            touched.add(index)
        }

        let changed = false
        for (const index of touched) changed = this.settle(index) || changed
        if (changed) this.landmassData.needsUpdate = true
    }

    private settle(landmass: number): boolean {
        const total = this.data.totals[landmass]
        const held = this.held[landmass]

        let holder: string | undefined
        let best = 0
        for (const [player, count] of held) {
            if (count > best) {
                best = count
                holder = player
            }
        }

        const share = total > 0 ? best / total : 0
        if (share < this.minimumShare || total < this.minimumTiles) holder = undefined

        const at = landmass * LANDMASS_TEXELS * 4
        const region = holder === undefined ? undefined : regions.get(holder)
        const opacity = region === undefined ? 0 : Math.pow(share, this.contrast)

        if (holder === this.painted[landmass] && Math.abs(opacity - this.rows[at + 12]) < 0.002) {
            return false
        }
        this.painted[landmass] = holder

        if (!region) {
            this.rows[at + 8] = 0
            this.rows[at + 9] = 0
            this.rows[at + 10] = 0
            this.rows[at + 11] = 0
            this.rows[at + 12] = 0
            return true
        }

        const aspect = region.width / region.height
        const reachU = this.data.frames[landmass * 5 + 3]
        const reachV = this.data.frames[landmass * 5 + 4]

        let halfU: number
        let halfV: number
        if (this.stretch && holder !== undefined && fits[holder]?.stretch && reachU > 0 && reachV > 0) {
            const wanted = reachU / reachV
            const pulled = Math.min(Math.max(wanted, aspect / MAX_STRETCH), aspect * MAX_STRETCH)
            halfV = Math.max(reachV, reachU / pulled)
            halfU = halfV * pulled
        } else if (this.fit === "contain") {
            halfV = Math.min(reachV, reachU / aspect)
            halfU = halfV * aspect
        } else {
            halfV = Math.max(reachV, reachU / aspect)
            halfU = halfV * aspect
        }

        if (halfV > MAX_FLAG_RADIUS) {
            halfU *= MAX_FLAG_RADIUS / halfV
            halfV = MAX_FLAG_RADIUS
        }

        this.rows[at + 3] = halfU
        this.rows[at + 7] = halfV

        const named = holder ?? ""
        this.rows[at + 14] = anchor(focusOf(named, 0), reachU / (2 * halfU))
        this.rows[at + 15] = anchor(1 - focusOf(named, 1), reachV / (2 * halfV))
        this.rows[at + 8] = region.x
        this.rows[at + 9] = region.y
        this.rows[at + 10] = region.width
        this.rows[at + 11] = region.height
        this.rows[at + 12] = opacity
        return true
    }

    dispose() {
        this.landmassData.dispose()
    }
}

const MINIMUM_TILES = 4

const MINIMUM_SHARE = 0.08

const CONTRAST = 1

const MAX_FLAG_RADIUS = 0.42

const MAX_STRETCH = 2.6

type FlagFit = {stretch: boolean, focus: number[]}
const fits: Record<string, FlagFit> = flagFit

const FOCUS_BIAS = 2

function focusOf(code: string, axis: 0 | 1): number {
    const focus = fits[code]?.focus?.[axis] ?? 0.5
    return Math.min(1, Math.max(0, 0.5 + (focus - 0.5) * FOCUS_BIAS))
}

function anchor(focus: number, half: number): number {
    return half >= 0.5 ? 0.5 : Math.min(Math.max(focus, half), 1 - half)
}
