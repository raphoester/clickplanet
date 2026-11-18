import * as THREE from "three"
import {type ChunkView, drawnRanges, showingChunks, type TileChunks} from "../../domain/tileChunks.ts"
import type {Span} from "../../domain/spans.ts"

// In screen half-heights: the pieces are found with this much to spare, and kept until the view has moved by it.
export const CULL_SLACK = 0.25

// The pieces in sight, and the view they were found for.
type Sighting = {
    ranges: Span[]
    right: THREE.Vector3
    up: THREE.Vector3
    back: THREE.Vector3
    sx: number, sy: number, tx: number, ty: number
    limb: number, reach: number, slack: number
}

type View = Omit<Sighting, "ranges" | "slack">

export class ChunkSight {
    private sighting: Sighting | undefined

    constructor(
        private readonly chunks: TileChunks,
        private readonly shown: Uint32Array,
        private readonly most: number,
        private readonly least: number,
    ) {}

    // The same ranges while they still hold, so a caller can tell that nothing changed.
    rangesFor(camera: THREE.Camera, toClip: ArrayLike<number>, limb: number, reach: number, slack: number): Span[] {
        const m = toClip
        const sx = Math.hypot(m[0], m[4], m[8]), sy = Math.hypot(m[1], m[5], m[9])
        const now: View = {
            right: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 0).normalize(),
            up: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 1).normalize(),
            back: new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 2).normalize(),
            sx, sy, tx: m[12] / sx, ty: m[13] / sy, limb, reach,
        }

        if (!this.sighting || !stillHolds(this.sighting, now)) {
            const {back} = now
            const view: ChunkView = {toClip: m, back: [back.x, back.y, back.z], limb: limb + slack, reach: reach + slack}
            const shown = showingChunks(this.chunks, view, this.shown)
            this.sighting = {...now, slack, ranges: drawnRanges(this.chunks, this.shown, shown, this.most, this.least)}
        }
        return this.sighting.ranges
    }
}

// The screen no larger, the pieces reaching no further, and the camera turned and moved by less than the slack.
function stillHolds(last: Sighting, now: View): boolean {
    const close = 1 - 1e-9
    if (now.sx < last.sx * close || now.sy < last.sy * close || now.limb * close > last.limb || now.reach * close > last.reach) return false
    return now.right.distanceTo(last.right) + Math.abs(now.tx - last.tx) <= last.slack
        && now.up.distanceTo(last.up) + Math.abs(now.ty - last.ty) <= last.slack
        && now.back.distanceTo(last.back) <= last.slack
}
