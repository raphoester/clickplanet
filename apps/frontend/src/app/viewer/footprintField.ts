import * as THREE from "three"
import {type FootprintFrame, Footprints, NO_FOOTPRINT, unitPositions} from "../../domain/footprints.ts"
import {joinSpans, spansOf} from "../../domain/spans.ts"
import {aspectOf, flagCodeOf, SPRITE_WIDTH} from "./atlas.ts"
import {detailOf, MAX_FLAG_RADIUS} from "./borderField.ts"
import type {Divisions} from "./pointSize.ts"
import {flagNeedOf, type ScreenView} from "./slotNeed.ts"
import {addNeed, type FlagNeed} from "./slotRanking.ts"

// The grounds of a framing as the territory layer paints them: each one's leader, drawn in the ground's own frame.
export type PaintedGrounds = {
    holderOf(ground: number): string | undefined
    sizeOf(ground: number): number
}

// Must match FOOTPRINT_COLUMNS, FOOTPRINT_ROWS in shaders/territory/fragment.glsl, and TILE_COLUMNS in tileField.ts.
export const FOOTPRINT_COLUMNS = 1024
const TILE_COLUMNS = 1024
// Footprints a framing holds, the none among them: rows of FOOTPRINT_COLUMNS, two texels each.
export const MOST_FOOTPRINTS = 32 * FOOTPRINT_COLUMNS
const FOOTPRINT_ROWS = MOST_FOOTPRINTS / FOOTPRINT_COLUMNS
const TEXELS = 2
const MAX_ROW_UPLOADS = 16

// Where each flag's own tiles lie, on each province and on each landmass: the frame the territory layer paints a flag
// in on the ground it does not lead, so a tile wears the part of its flag where it is in its flag's own ground.
// `tileData` holds each tile's province footprint at its cell's texel; `frames` a row a footprint, the provinces' then
// the landmasses', each province footprint naming its landmass's.
export class FootprintField {
    readonly tileData: THREE.DataTexture
    readonly frames: THREE.DataTexture

    private readonly tileValues: Uint16Array
    private readonly frameValues: Float32Array
    private readonly changedTileRows = new Set<number>()
    private readonly changedFrameRows = new Set<number>()
    private readonly unit: Float32Array
    private provinces: Footprints | undefined
    private landmasses: Footprints | undefined
    private readonly parentOf = new Uint16Array(MOST_FOOTPRINTS)
    private readonly flagOfTile: Uint16Array

    constructor(
        positions: Float32Array,
        private readonly texels: Uint32Array,
        private readonly flags: number,
    ) {
        this.unit = unitPositions(positions)
        this.flagOfTile = new Uint16Array(texels.length)
        const rows = Math.ceil(texels.length / TILE_COLUMNS)
        this.tileValues = new Uint16Array(TILE_COLUMNS * rows)
        this.tileData = new THREE.DataTexture(this.tileValues, TILE_COLUMNS, rows, THREE.RedIntegerFormat, THREE.UnsignedShortType)
        this.tileData.internalFormat = "R16UI"
        this.tileData.needsUpdate = true

        this.frameValues = new Float32Array(FOOTPRINT_COLUMNS * TEXELS * FOOTPRINT_ROWS * 2 * 4)
        this.frames = new THREE.DataTexture(this.frameValues, FOOTPRINT_COLUMNS * TEXELS, FOOTPRINT_ROWS * 2, THREE.RGBAFormat, THREE.FloatType)
        this.frames.minFilter = THREE.NearestFilter
        this.frames.magFilter = THREE.NearestFilter
        this.frames.needsUpdate = true
    }

    // Each tile's province and each province's landmass: the footprints are counted again from the flags they hold.
    setGrounds(province: Uint16Array, landmassOf: Uint16Array, provinceCount: number, landmassCount: number) {
        const size = province.length
        const landmass = new Uint16Array(size)
        for (let i = 0; i < size; i++) landmass[i] = landmassOf[province[i]] ?? 0
        this.provinces = new Footprints(this.unit, province, provinceCount, this.flags, MOST_FOOTPRINTS)
        this.landmasses = new Footprints(this.unit, landmass, landmassCount, this.flags, MOST_FOOTPRINTS)
        this.frameValues.fill(0)
        this.parentOf.fill(0)
        this.tileValues.fill(0)
        for (let tile = 0; tile < size; tile++) {
            const flag = this.flagOfTile[tile]
            if (flag !== 0) this.place(tile, flag)
        }
        this.frames.needsUpdate = true
        this.tileData.needsUpdate = true
        this.changedTileRows.clear()
        this.writeFrames()
        this.changedFrameRows.clear()
    }

    // By tile index (tile id - 1) and flag number, 0 none.
    setFlags(tiles: readonly number[], flags: readonly number[]) {
        tiles.forEach((tile, i) => {
            this.flagOfTile[tile] = flags[i]
            if (this.provinces) this.place(tile, flags[i])
        })
    }

    // What each flag painted over its own footprint on a ground in view asks of the slots: its frame's width on screen,
    // and its share of the ground's land on screen. A ground's leader is drawn in the ground's frame: BorderField's.
    flagNeeds(divisions: Divisions, view: ScreenView, areas: Float32Array, grounds: PaintedGrounds, leastRadius: number): Map<string, FlagNeed> {
        const needs = new Map<string, FlagNeed>()
        const footprints = divisions === "provinces" ? this.provinces : this.landmasses
        footprints?.forEachFootprint((id, ground, flag) => {
            const area = areas[ground] ?? 0
            const code = flagCodeOf(flag)
            if (area <= 0 || code === undefined || code === grounds.holderOf(ground)) return
            const aspect = aspectOf(code)
            const frame = footprints.frameOf(id)
            const size = grounds.sizeOf(ground)
            if (aspect === undefined || frame === undefined || size <= 0) return
            const halfV = coverOf(frame, aspect, leastRadius)
            const [x, y, z] = frame.centre
            addNeed(needs, code, {
                need: flagNeedOf({x, y, z}, halfV * aspect, halfV, aspect, view),
                area: area * footprints.countOf(id) / size,
                aspect,
                spriteWidth: SPRITE_WIDTH,
                detail: detailOf(code),
            })
        })
        return needs
    }

    upload(renderer: THREE.WebGLRenderer) {
        if (!this.provinces) return
        this.writeFrames()
        copyRows(renderer, this.tileData, this.changedTileRows, TILE_COLUMNS)
        copyRows(renderer, this.frames, this.changedFrameRows, FOOTPRINT_COLUMNS * TEXELS)
    }

    dispose() {
        this.tileData.dispose()
        this.frames.dispose()
    }

    private place(tile: number, flag: number) {
        const landmass = this.landmasses!.set(tile, flag)
        const province = this.provinces!.set(tile, flag)
        if (province !== NO_FOOTPRINT && this.parentOf[province] !== landmass) {
            this.parentOf[province] = landmass
            this.writeFrame(province, 0)
        }
        const texel = this.texels[tile]
        if (this.tileValues[texel] !== province) {
            this.tileValues[texel] = province
            this.changedTileRows.add(Math.floor(texel / TILE_COLUMNS))
        }
    }

    private writeFrames() {
        for (const id of this.provinces!.takeChanged()) this.writeFrame(id, 0)
        for (const id of this.landmasses!.takeChanged()) this.writeFrame(id, FOOTPRINT_ROWS)
    }

    // Texel 0: the centre and the half extent east; texel 1: the half extent north, the landmass's footprint (provinces
    // alone), its tiles. A footprint with no tile is all 0, which the shader reads as no frame.
    private writeFrame(id: number, firstRow: number) {
        const footprints = firstRow === 0 ? this.provinces! : this.landmasses!
        const row = firstRow + Math.floor(id / FOOTPRINT_COLUMNS)
        const at = (row * FOOTPRINT_COLUMNS * TEXELS + (id % FOOTPRINT_COLUMNS) * TEXELS) * 4
        const frame = footprints.frameOf(id)
        const v = this.frameValues
        if (!frame) {
            v.fill(0, at, at + TEXELS * 4)
        } else {
            v.set(frame.centre, at)
            v[at + 3] = frame.halfEast
            v[at + 4] = frame.halfNorth
            v[at + 5] = firstRow === 0 ? this.parentOf[id] : 0
            v[at + 6] = 0
            v[at + 7] = footprints.countOf(id)
        }
        this.changedFrameRows.add(row)
    }
}

// As coverOf in the territory fragment shader: the half height of the flag over a footprint, from its middle.
function coverOf({halfEast, halfNorth}: FootprintFrame, aspect: number, leastRadius: number): number {
    return Math.min(MAX_FLAG_RADIUS, Math.max(leastRadius, halfNorth, halfEast / aspect))
}

function copyRows(renderer: THREE.WebGLRenderer, texture: THREE.DataTexture, changed: Set<number>, width: number) {
    if (changed.size === 0) return
    const rows = Uint32Array.from(changed).sort()
    changed.clear()
    for (const {start, count} of joinSpans(spansOf(rows), MAX_ROW_UPLOADS)) {
        renderer.copyTextureToTexture(
            texture,
            texture,
            new THREE.Box2(new THREE.Vector2(0, start), new THREE.Vector2(width, start + count)),
            new THREE.Vector2(0, start),
        )
    }
}
