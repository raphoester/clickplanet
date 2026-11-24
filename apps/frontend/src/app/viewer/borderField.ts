import {BoardScale, scaledCount} from "../../domain/boardScale.ts"
import type {BoardAsset} from "../../domain/board.ts"
import {bordersHashOf, decodeProvinces, landmassProvinces, type Provinces} from "../../domain/provinces.ts"
import * as THREE from "three"
import {flagNumberOf, pictureOf, regionOf} from "./atlas.ts"
import {flagNeedOf, type ScreenView} from "./slotNeed.ts"
import {addNeed, type FlagNeed} from "./slotRanking.ts"
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

// The board's provinces, read beside its borders; a board with no provinces file has its landmasses as provinces.
export async function loadProvinces(board: BoardAsset, borders: Promise<BorderData>, signal?: AbortSignal): Promise<Provinces> {
    const borderHash = bordersHashOf(board.files.borders)
    if (!board.files.provinces) {
        const {assignment, frames, totals} = await borders
        return landmassProvinces(borderHash, assignment, frames, totals)
    }
    const url = board.urls.provinces!
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    const provinces = decodeProvinces(await response.arrayBuffer(), board.tiles)
    if (provinces.borders !== borderHash) throw new Error(`${url} was cut on other borders than ${board.files.borders}`)
    return provinces
}

export function countryOfTile(data: BorderData, tile: number): string | undefined {
    const index = data.assignment[tile - 1]
    return index ? data.codes[index] : undefined
}

// The grounds a flag is painted over, landmasses or provinces: each one's frame (centre x, y, z, half extents east and
// north) and its tiles, 0 none.
export type Grounds = {frames: Float32Array, totals: Uint32Array}

// A row a ground: some players' GPUs take no texture taller (docs/tile-layout.md).
export const MOST_GROUNDS = 4096

export class BorderField {
    readonly texture: THREE.DataTexture
    readonly count: number

    private readonly rows: Float32Array
    private readonly painted: (string | undefined)[]
    private readonly leaders: ({holder: string, held: number} | undefined)[]

    constructor(
        private readonly data: Grounds,
        private readonly minimumShare = MINIMUM_SHARE,
        private readonly minimumTiles = MINIMUM_TILES,
        private readonly stretch = true,
        private readonly fit: "cover" | "contain" = "cover",
    ) {
        this.count = data.totals.length
        if (this.count > MOST_GROUNDS) throw new Error(`${this.count} grounds, more rows than a texture takes (${MOST_GROUNDS})`)
        this.painted = new Array(this.count)
        this.leaders = new Array(this.count)

        this.rows = new Float32Array(LANDMASS_TEXELS * this.count * 4)
        for (let piece = 1; piece < this.count; piece++) {
            const at = piece * LANDMASS_TEXELS * 4
            this.rows[at] = data.frames[piece * 5]
            this.rows[at + 1] = data.frames[piece * 5 + 1]
            this.rows[at + 2] = data.frames[piece * 5 + 2]
            this.rows[at + 4] = data.frames[piece * 5 + 3]
            this.rows[at + 5] = data.frames[piece * 5 + 4]
            this.rows[at + 6] = 0
        }

        this.texture = new THREE.DataTexture(this.rows, LANDMASS_TEXELS, this.count, THREE.RGBAFormat, THREE.FloatType)
        this.texture.minFilter = THREE.NearestFilter
        this.texture.magFilter = THREE.NearestFilter
        this.texture.needsUpdate = true
    }

    holderOf(landmass: number): string | undefined {
        return this.painted[landmass]
    }

    sizeOf(landmass: number): number {
        return this.data.totals[landmass] ?? 0
    }

    // Each painted flag on screen, over the landmasses it leads: its widest frame and all their land.
    flagNeeds(view: ScreenView, areas: Float32Array): Map<string, FlagNeed> {
        const needs = new Map<string, FlagNeed>()
        for (let landmass = 1; landmass < this.count; landmass++) {
            const holder = this.painted[landmass]
            const at = landmass * LANDMASS_TEXELS * 4
            if (holder === undefined || this.rows[at + 10] <= 0) continue
            const aspect = this.rows[at + 10] / this.rows[at + 11]
            const area = areas[landmass]
            const centre = {x: this.rows[at], y: this.rows[at + 1], z: this.rows[at + 2]}
            const need = area > 0 ? flagNeedOf(centre, this.rows[at + 3], this.rows[at + 7], aspect, view) : 0
            addNeed(needs, holder, {need, area, aspect, spriteWidth: this.rows[at + 10], detail: detailOf(holder)})
        }
        return needs
    }

    // Gives each landmass the flag that leads it, at the share of it that flag holds: fitted to its ground, drawn large.
    setRows(landmasses: Iterable<number>, leaderOf: (landmass: number) => {holder: string, held: number} | undefined) {
        let changed = false
        for (const landmass of landmasses) {
            if (landmass >= 1 && landmass < this.count) changed = this.settle(landmass, leaderOf(landmass)) || changed
        }
        if (changed) this.texture.needsUpdate = true
    }

    // A flag's picture reached the atlas: each ground it leads is framed again at the picture's shape.
    showFlag(code: string) {
        let changed = false
        for (let landmass = 1; landmass < this.count; landmass++) {
            if (this.painted[landmass] !== code) continue
            this.painted[landmass] = undefined
            changed = this.settle(landmass, this.leaders[landmass]) || changed
        }
        if (changed) this.texture.needsUpdate = true
    }

    private settle(landmass: number, leader: {holder: string, held: number} | undefined): boolean {
        this.leaders[landmass] = leader
        const total = this.data.totals[landmass]

        let holder = leader?.holder
        const share = total > 0 ? (leader?.held ?? 0) / total : 0
        if (share < this.minimumShare || total < this.minimumTiles) holder = undefined

        const at = landmass * LANDMASS_TEXELS * 4
        const region = holder === undefined ? undefined : regionOf(holder)
        const opacity = region === undefined ? 0 : share

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
            this.rows[at + 13] = 0
            return true
        }

        const aspect = region.width / region.height
        const reachU = this.data.frames[landmass * 5 + 3]
        const reachV = this.data.frames[landmass * 5 + 4]

        let halfU: number
        let halfV: number
        if (this.stretch && holder !== undefined && fitOf(holder)?.stretch && reachU > 0 && reachV > 0) {
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
        this.rows[at + 13] = flagNumberOf(named)
        return true
    }

    dispose() {
        this.texture.dispose()
    }
}

// A landmass of fewer tiles of earth-1 is too small to paint a flag in: as much ground on another board.
const MINIMUM_TILES = 4

export function leastPaintedTiles(scale: BoardScale): number {
    return scaledCount(MINIMUM_TILES, scale)
}

const MINIMUM_SHARE = 0.08

export const MAX_FLAG_RADIUS = 0.42

const MAX_STRETCH = 2.6

type FlagFit = {stretch: boolean, focus: number[], detail?: number}
const fits: Record<string, FlagFit> = flagFit

const FOCUS_BIAS = 2

// A sold flag's picture says what flagFit.json says of a built one.
function fitOf(code: string): FlagFit | undefined {
    const picture = pictureOf(code)
    return picture ? {stretch: picture.stretch, focus: [picture.focusX, picture.focusY]} : fits[code]
}

// A sold flag's picture is not measured: it counts as the median flag.
export function detailOf(code: string): number {
    return fitOf(code)?.detail ?? 1
}

function focusOf(code: string, axis: 0 | 1): number {
    const focus = fitOf(code)?.focus?.[axis] ?? 0.5
    return Math.min(1, Math.max(0, 0.5 + (focus - 0.5) * FOCUS_BIAS))
}

function anchor(focus: number, half: number): number {
    return half >= 0.5 ? 0.5 : Math.min(Math.max(focus, half), 1 - half)
}
