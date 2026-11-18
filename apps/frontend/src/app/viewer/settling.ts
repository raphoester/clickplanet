export type Viewpoint = {x: number, y: number, z: number, scale: number}

export type Motion = "settled" | "moving" | "calm"

const SAMPLE_MS = 250

const STILL_TURN = 0.004
const STILL_SCALE = 0.005

// Radians a second: the idle spin turns 0.21, a drag several.
const DRIFT_SPEED = 0.5

const DRIFT_TURN = 0.25

export class Settling {
    private sample: {at: number, viewpoint: Viewpoint} | undefined
    private ranked: Viewpoint | undefined
    private motion: Motion = "calm"

    tick(at: number, viewpoint: Viewpoint): Motion {
        if (this.sample === undefined) {
            this.sample = {at, viewpoint}
            return this.motion
        }
        const seconds = (at - this.sample.at) / 1000
        if (seconds * 1000 < SAMPLE_MS) return this.motion

        const turned = angleBetween(this.sample.viewpoint, viewpoint)
        const steady = zoomed(this.sample.viewpoint, viewpoint) <= STILL_SCALE
        this.sample = {at, viewpoint}
        this.motion = steady && turned / seconds <= DRIFT_SPEED ? "calm" : "moving"
        if (this.motion === "moving") return this.motion

        const ranked = this.ranked
        const still = turned <= STILL_TURN
        const moved = ranked === undefined || zoomed(ranked, viewpoint) > STILL_SCALE
            || angleBetween(ranked, viewpoint) > (still ? STILL_TURN : DRIFT_TURN)
        if (!moved) return this.motion
        this.ranked = viewpoint
        return "settled"
    }
}

function angleBetween(a: Viewpoint, b: Viewpoint): number {
    const cross = Math.hypot(a.y * b.z - a.z * b.y, a.z * b.x - a.x * b.z, a.x * b.y - a.y * b.x)
    return Math.atan2(cross, a.x * b.x + a.y * b.y + a.z * b.z)
}

function zoomed(a: Viewpoint, b: Viewpoint): number {
    return Math.abs(Math.log(b.scale / a.scale))
}
