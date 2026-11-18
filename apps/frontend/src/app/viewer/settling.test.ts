import {describe, expect, it} from "vitest"
import {type Motion, Settling, type Viewpoint} from "./settling.ts"

const at = (longitude: number, scale = 1000): Viewpoint => ({x: Math.sin(longitude), y: 0, z: Math.cos(longitude), scale})

// Ticks every 16 ms for `ms`, turning `speed` radians a second from `from`: the motions seen, in order, once each.
function run(settling: Settling, start: number, ms: number, from: number, speed: number, scale = 1000) {
    const seen: Motion[] = []
    for (let t = 0; t <= ms; t += 16) {
        const motion = settling.tick(start + t, at(from + speed * t / 1000, scale))
        if (motion === "settled" || seen[seen.length - 1] !== motion) seen.push(motion)
    }
    return seen
}

describe("Settling", () => {
    it("settles once on a still camera, and not again while it stays", () => {
        const settling = new Settling()

        expect(run(settling, 0, 2000, 0, 0)).toEqual(["calm", "settled", "calm"])
    })

    it("does not settle while the camera moves fast, and settles once it stops", () => {
        const settling = new Settling()
        run(settling, 0, 1000, 0, 0)

        expect(run(settling, 1000, 1000, 0, 3)).toEqual(["calm", "moving"])
        expect(run(settling, 2000, 1000, 3, 0)).toEqual(["moving", "settled", "calm"])
    })

    it("settles every quarter radian while the camera turns slowly, as in the idle spin", () => {
        const settling = new Settling()
        const seen = run(settling, 0, 10_000, 0, 0.21)

        expect(seen.filter((motion) => motion === "settled").length).toBeGreaterThanOrEqual(7)
        expect(seen).not.toContain("moving")
    })

    it("settles after a zoom, at the new scale", () => {
        const settling = new Settling()
        run(settling, 0, 1000, 0, 0)

        const seen: Motion[] = []
        for (let t = 0; t <= 2000; t += 16) seen.push(settling.tick(1000 + t, at(0, t < 500 ? 1000 + t * 4 : 3000)))

        expect(seen).toContain("moving")
        expect(seen.filter((motion) => motion === "settled")).toHaveLength(1)
        expect(seen.lastIndexOf("moving")).toBeLessThan(seen.indexOf("settled"))
    })
})
