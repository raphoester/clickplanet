import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {EARTH_1_SCALE, EARTH_1_SPACING} from "../../domain/boardScale.ts"
import {
    BURST_SECONDS,
    choreographCures,
    choreographSpread,
    createVirusSlime,
    doom,
    POP_AT,
    RECOIL_SECONDS,
    SETTLED,
    type Slime,
    SLIME_IMPACT,
    slimeLifetime,
    slimeLook,
    sourceOf,
    SPLAT_SECONDS,
} from "./virusSlime.ts"

// Tile 1 at the pole, tiles 2 to 7 one spacing round it, tile 8 ten spacings off.
const around = Array.from({length: 6}, (_, i) => {
    const angle = (i * Math.PI) / 3
    return new THREE.Vector3(Math.cos(angle) * EARTH_1_SPACING, Math.sin(angle) * EARTH_1_SPACING, 1).normalize()
})
const far = new THREE.Vector3(10 * EARTH_1_SPACING, 0, 1).normalize()
const positions = new Float32Array([0, 0, 1, ...around.flatMap((p) => [p.x, p.y, p.z]), far.x, far.y, far.z])

const slime = (kind: Slime["kind"], start = 0): Slime => ({kind, from: 1, to: 2, start, seed: 1})
const fixed = () => 0.5

describe("slimeLook", () => {
    it("stretches a tendril from its source and lands on its target at the impact", () => {
        const ooze = slime("ooze")
        expect(slimeLook(ooze, 0)!.along).toBe(0)
        expect(slimeLook(ooze, SLIME_IMPACT * 0.6)!.along).toBeGreaterThan(0.3)
        expect(slimeLook(ooze, SLIME_IMPACT * 0.6)!.neck).toBeGreaterThan(0)
        expect(slimeLook(ooze, SLIME_IMPACT)!.along).toBe(1)
    })

    it("splats over its tile at the impact, then snaps from its source and sinks in", () => {
        const ooze = slime("ooze")
        const impact = slimeLook(ooze, SLIME_IMPACT)!
        expect(impact.head).toBeGreaterThan(SETTLED * 1.3)
        expect(impact.drop).toBeGreaterThan(0)
        expect(impact.flash).toBeGreaterThan(0)

        const letGo = slimeLook(ooze, SLIME_IMPACT + 0.45)!
        expect(letGo.neck).toBe(0)
        expect(letGo.source).toBe(0)
        expect(slimeLook(ooze, SLIME_IMPACT + 0.6)!.drop).toBe(0)

        const end = slimeLook(ooze, SLIME_IMPACT + SPLAT_SECONDS - 1e-3)!
        expect(end.alpha).toBeLessThan(0.01)
        expect(end.head).toBeLessThan(SETTLED)
        expect(slimeLook(ooze, SLIME_IMPACT + SPLAT_SECONDS)).toBeUndefined()
    })

    it("bursts a tendril's goo as it lands on a tile cured at once, and still lets go of its source", () => {
        const ooze = {...slime("ooze"), doomed: true}
        expect(slimeLook(ooze, SLIME_IMPACT / 2)).toEqual(slimeLook(slime("ooze"), SLIME_IMPACT / 2))

        const landed = slimeLook(ooze, SLIME_IMPACT + 0.1)!
        expect(landed.drop).toBeGreaterThan(0)
        expect(landed.spray).toBe(1)
        expect(landed.neck).toBeGreaterThan(0)
        expect(slimeLook(ooze, SLIME_IMPACT + 0.45)!.neck).toBe(0)

        const dry = slimeLook(ooze, slimeLifetime("ooze", false, true) - 1e-3)!
        expect(dry.dead).toBe(1)
        expect(dry.alpha).toBeLessThan(0.01)
    })

    it("shows no tendril before it sets out", () => {
        expect(slimeLook(slime("ooze", 0.05), 0.02)).toBeUndefined()
        expect(slimeLook(slime("ooze", 0.05), 0.05)!.along).toBe(0)
    })

    it("boils up out of the tile a virus was used on, with no source", () => {
        const erupt = slime("erupt")
        const rising = [0.1, 0.2, 0.3].map((t) => slimeLook(erupt, t)!.head)
        expect(rising[0]).toBeLessThan(rising[1])
        expect(rising[1]).toBeLessThan(rising[2])
        for (const t of [0.1, SLIME_IMPACT, SLIME_IMPACT + 0.2]) {
            const look = slimeLook(erupt, t)!
            expect(look.along).toBe(1)
            expect(look.neck).toBe(0)
            expect(look.source).toBe(0)
        }
        expect(slimeLook(erupt, SLIME_IMPACT)!.spray).toBe(1)
    })

    it("reaches halfway to a shield and pulls back into its tile", () => {
        const recoil = slime("recoil")
        expect(slimeLook(recoil, SLIME_IMPACT)!.along).toBeCloseTo(0.5, 9)
        expect(slimeLook(recoil, SLIME_IMPACT + RECOIL_SECONDS)!.along).toBeCloseTo(0, 9)
        expect(slimeLook(recoil, slimeLifetime("recoil") - 1e-3)!.alpha).toBeLessThan(0.01)
    })

    it("swells a cured tile's goo whole, then bursts it into droplets and dries it to a flat husk", () => {
        const burst = slime("burst")
        const swelling = slimeLook(burst, POP_AT / 2)!
        expect(swelling.head).toBeGreaterThan(SETTLED)
        expect(swelling.drop).toBe(0)
        expect(swelling.dead).toBe(0)

        const popped = slimeLook(burst, POP_AT + 0.1)!
        expect(popped.drop).toBeGreaterThan(0)
        expect(popped.drops).toBeGreaterThan(swelling.head)
        expect(popped.spray).toBe(1)

        const dry = slimeLook(burst, BURST_SECONDS - 1e-3)!
        expect(dry.dead).toBe(1)
        expect(dry.gloss).toBe(0)
        expect(dry.alpha).toBeLessThan(0.01)
        expect(slimeLook(burst, BURST_SECONDS)).toBeUndefined()
    })

    it("holds a cured tile's goo whole until its burst starts: its icon is already gone", () => {
        const look = slimeLook(slime("burst", 0.1), 0)!
        expect(look.alpha).toBeGreaterThan(0.9)
        expect(look.head).toBeGreaterThanOrEqual(SETTLED)
    })

    it("moves nothing with less motion: the goo fades in on its tile and a cured one dries", () => {
        for (const t of [0.1, SLIME_IMPACT * 0.9]) expect(slimeLook(slime("ooze"), t, true)).toBeUndefined()
        for (const t of [SLIME_IMPACT, SLIME_IMPACT + 0.5]) {
            const look = slimeLook(slime("ooze"), t, true)!
            expect(look.along).toBe(1)
            expect(look.drop).toBe(0)
            expect(look.wobble).toBe(0)
            expect(look.neck).toBe(0)
        }
        expect(slimeLifetime("recoil", true)).toBe(0)
        const dry = slimeLook(slime("burst"), BURST_SECONDS * 0.9, true)!
        expect(dry.drop).toBe(0)
        expect(dry.dead).toBe(1)
    })
})

describe("sourceOf", () => {
    const reach = 4 * EARTH_1_SPACING

    it("is the virus's nearest tile", () => {
        expect(sourceOf(1, [8, 3], positions, reach)).toBe(3)
    })

    it("is none past the reach, nor the tile itself", () => {
        expect(sourceOf(1, [8], positions, reach)).toBeUndefined()
        expect(sourceOf(1, [1], positions, reach)).toBeUndefined()
    })
})

describe("choreographSpread", () => {
    it("erupts out of the tile a virus was used on", () => {
        const [use, ...rest] = choreographSpread({taken: [1], struck: []}, [], positions, EARTH_1_SCALE, fixed)
        expect(rest).toHaveLength(0)
        expect(use).toMatchObject({kind: "erupt", from: 1, to: 1, start: 0})
    })

    it("oozes onto each tile a tick took from the nearest tile the virus holds, and recoils from each shield", () => {
        const slimes = choreographSpread({taken: [3, 4], struck: [{tile: 6, shields: 1}]}, [1, 2], positions, EARTH_1_SCALE, fixed)
        expect(slimes.map(({kind, from, to}) => ({kind, from, to}))).toEqual([
            {kind: "ooze", from: 2, to: 3},
            {kind: "ooze", from: 1, to: 4},
            {kind: "recoil", from: 1, to: 6},
        ])
    })

    it("draws no recoil from a shield with none of the virus's tiles near", () => {
        expect(choreographSpread({taken: [], struck: [{tile: 8, shields: 2}]}, [1], positions, EARTH_1_SCALE, fixed)).toEqual([])
    })
})

describe("choreographCures", () => {
    it("bursts each tile, one alone at once", () => {
        expect(choreographCures([5], fixed)).toEqual([{kind: "burst", from: 5, to: 5, start: 0, seed: 50}])
        const many = choreographCures([2, 3, 4], () => 0.4)
        expect(many.map(({to}) => to)).toEqual([2, 3, 4])
        for (const burst of many) expect(burst.start).toBeGreaterThan(0)
    })
})

describe("doom", () => {
    it("bursts the goo landing on a cured tile there, and leaves the other tiles to burst on their own", () => {
        const landing = {...slime("ooze"), to: 3, startedAt: 10}
        const sunk = {...slime("ooze"), to: 4, startedAt: 8}
        const elsewhere = {...slime("ooze"), to: 5, startedAt: 10}
        const left = doom([landing, sunk, elsewhere], [3, 4, 6], 10 + SLIME_IMPACT)
        expect(left).toEqual([4, 6])
        expect(landing.doomed).toBe(true)
        expect(sunk.doomed).toBeUndefined()
        expect(elsewhere.doomed).toBeUndefined()
    })

    it("bursts no shield's recoil", () => {
        const recoil = {...slime("recoil"), to: 3, startedAt: 10}
        expect(doom([recoil], [3], 10.2)).toEqual([3])
        expect(recoil.doomed).toBeUndefined()
    })
})

describe("createVirusSlime", () => {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1)
    camera.zoom = 20

    it("asks for frames while goo plays, the frame it ends on included, then none", () => {
        const slimes = createVirusSlime(positions, EARTH_1_SCALE, {value: 0.012})
        expect(slimes.update(0, camera, 1000, 1)).toBe(false)

        slimes.cure([2])
        expect(slimes.update(1, camera, 1000, 1)).toBe(true)
        expect(slimes.update(1 + BURST_SECONDS / 2, camera, 1000, 1)).toBe(true)
        expect(slimes.update(1 + BURST_SECONDS + 0.01, camera, 1000, 1)).toBe(true)
        expect(slimes.update(1 + BURST_SECONDS + 0.02, camera, 1000, 1)).toBe(false)
        slimes.dispose()
    })

    it("draws one instance per goo playing", () => {
        const slimes = createVirusSlime(positions, EARTH_1_SCALE, {value: 0.012})
        slimes.spread({taken: [3, 4], struck: []}, [1, 2])
        slimes.update(0, camera, 1000, 1)
        const mesh = slimes.object as THREE.Mesh<THREE.InstancedBufferGeometry>
        expect(mesh.geometry.instanceCount).toBe(2)
        slimes.dispose()
    })
})
