// SCRATCH — R&D prototype of the zoomed-out view, keyed on real borders.
//
// Every tile sits inside one *landmass*: a country's tiles split into the
// separate pieces of land they actually form (Natural Earth 1:50m, resolved
// offline — neither borders nor tiles move, so tile → landmass and the frame a
// flag is painted in are both static tables).
//
// The unit is a landmass and not a country because a flag belongs to a piece of
// ground. France is mainland, Corsica, Guiana, Réunion; one frame spanning all
// of them would stretch the tricolour across half the planet and paint nothing
// recognisable anywhere. Mainland France finished is mainland France flying the
// flag, whatever is happening in the Pacific.
//
// Each landmass flies the flag of whoever holds the most of it, painted across
// it, following the surface, cropped by its own coastline — at the opacity of
// how much of it they hold. Half of Poland is a half-transparent tricolour over
// the Earth underneath; all of Poland is a solid one. So the globe says who is
// winning where *and* how settled it is, in one reading, and no country ever
// needs a colour invented for it.
import * as THREE from "three"
import type {OwnerChange} from "../../domain/tileOwnership.ts"
import {regions} from "./atlas.ts"
import flagFit from "../../../static/countries/flagFit.json"

// Per landmass: centre + half-width, east axis + half-height, atlas region, and
// the leader's share plus how far the landmass itself reaches. A zero-width
// region is a landmass nobody holds.
const LANDMASS_TEXELS = 4

export type BorderData = {
    codes: string[]
    assignment: Uint16Array
    frames: Float32Array
    totals: Uint32Array
}

export async function loadBorders(url: string): Promise<BorderData> {
    const buffer = await (await fetch(url)).arrayBuffer()
    const headerBytes = new DataView(buffer).getUint32(0, true)
    const header = JSON.parse(new TextDecoder().decode(new Uint8Array(buffer, 4, headerBytes)))
    let at = 4 + headerBytes

    const assignment = new Uint16Array(buffer, at, header.tiles)
    at += header.tiles * 2
    const frames = new Float32Array(buffer, at, header.codes.length * 5)
    at += header.codes.length * 5 * 4
    const totals = new Uint32Array(buffer, at, header.codes.length)

    return {codes: header.codes, assignment, frames, totals}
}

export class BorderField {
    readonly landmassData: THREE.DataTexture
    readonly landmassCount: number

    private readonly held: Map<string, number>[]
    private readonly ownerOf: (string | undefined)[]
    private readonly rows: Float32Array
    private readonly painted: (string | undefined)[]

    /**
     * @param minimumTiles a landmass smaller than this never paints. A third of
     *        them are a single tile, and at any zoom where you could see such a
     *        flag the tile is already drawing its owner's flag by itself.
     * @param minimumShare a leader holding less than this paints nothing, so a
     *        landmass barely touched stays bare Earth rather than a ghost of a
     *        flag.
     * @param stretch off, every flag keeps its own proportions and is cropped.
     * @param contrast bends the share → opacity curve. Straight share is too
     *        generous when one player holds ~70% of the planet: everything goes
     *        nearly solid and the globe is wall-to-wall flags again. Cubed, a
     *        70% hold is a ghost and only a country somebody actually finished
     *        comes out vivid.
     */
    constructor(
        private readonly data: BorderData,
        tileCount: number,
        private readonly minimumShare = 0.08,
        private readonly contrast = 1,
        private readonly minimumTiles = 4,
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

            // How far the landmass itself reaches, so the shader can tell how
            // big it currently is on screen. Static, unlike the share.
            this.rows[at + 13] = Math.max(data.frames[piece * 5 + 3], data.frames[piece * 5 + 4])
        }

        this.landmassData = new THREE.DataTexture(
            this.rows, LANDMASS_TEXELS, this.landmassCount, THREE.RGBAFormat, THREE.FloatType,
        )
        this.landmassData.minFilter = THREE.NearestFilter
        this.landmassData.magFilter = THREE.NearestFilter
        this.landmassData.needsUpdate = true
    }

    /** Whoever holds the most of this landmass. */
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

        // Fitting a flag to a shape nobody chose. Keeping its proportions means
        // covering the landmass and letting the coastline crop the rest, which
        // is right for a flag carrying a device — but on a narrow country it
        // leaves only the middle of the flag showing, and the middle of a
        // tricolour is one band. Portugal and Sri Lanka held by France came out
        // plain white.
        //
        // A flag that is only bands can be pulled to the country's own shape
        // instead and still say what it is, so it is, up to the point where the
        // bands stop reading as a flag.
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
            // The whole flag, fitted inside the country. Nothing is cropped, so
            // a device at the hoist survives — at the price of the flag no
            // longer reaching the coast, which the edge pixels then extend.
            halfV = Math.min(reachV, reachU / aspect)
            halfU = halfV * aspect
        } else {
            halfV = Math.max(reachV, reachU / aspect)
            halfU = halfV * aspect
        }

        // The cap is on the flag's height, and the width follows from it.
        // Capping whichever side happens to be larger instead shrinks a wide
        // country's flag by its own aspect ratio — Antarctica's came out a
        // quarter of the area, a rectangle adrift in the middle of the
        // continent rather than the continent wearing it.
        if (halfV > MAX_FLAG_RADIUS) {
            halfU *= MAX_FLAG_RADIUS / halfV
            halfV = MAX_FLAG_RADIUS
        }

        this.rows[at + 3] = halfU
        this.rows[at + 7] = halfV

        // A narrow country sees a vertical slice of its flag, a wide one a
        // horizontal band. Taken from the middle, the slice misses whatever the
        // flag is actually known by, so it is taken around that instead.
        const named = holder ?? ""
        this.rows[at + 14] = anchor(focusOf(named, 0), reachU / (2 * halfU))
        this.rows[at + 15] = anchor(focusOf(named, 1), reachV / (2 * halfV))
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

// How tall a painted flag may get. The Russian and Antarctic mainlands reach
// far enough around the globe that a flag stretched over the whole of one stops
// reading as a flag, so past this it stays a big flag inside the landmass
// instead. It bounds the height only — the width follows the flag's own
// proportions and may exceed it.
const MAX_FLAG_RADIUS = 0.42

// How far a stretchable flag may be pulled from its own proportions before the
// rest is taken out of the crop instead. Past this even bands stop reading as a
// flag and start reading as stripes of paint.
const MAX_STRETCH = 2.6

type FlagFit = {stretch: boolean, focus: number[]}
const fits: Record<string, FlagFit> = flagFit

// The measure is a centre of mass over every line of the flag, so it lands
// nearer the middle than the eye does — most of a flag is ordinary, and it all
// pulls. Doubling the offset puts the crop where a person would put it, and
// leaves a flag with nothing to aim at exactly where it was.
const FOCUS_BIAS = 2

function focusOf(code: string, axis: 0 | 1): number {
    const focus = fits[code]?.focus?.[axis] ?? 0.5
    return Math.min(1, Math.max(0, 0.5 + (focus - 0.5) * FOCUS_BIAS))
}

// Where a crop of the given half-width should be centred: on what names the
// flag, pushed back inside it if that would hang off the edge.
function anchor(focus: number, half: number): number {
    return half >= 0.5 ? 0.5 : Math.min(Math.max(focus, half), 1 - half)
}
