import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    BOOST_LIFETIME_SECONDS,
    BOOST_STREAKS,
    BOOST_TRAVEL_SECONDS,
    BOOST_WAVES_AT,
    choreographBoost,
    choreographSpread,
    createBonusClickEffects,
    type Spark,
    sparkLook,
    SPREAD_LIFETIME_SECONDS,
    SPREAD_THROW_FROM,
    SPREAD_THROW_STAGGER,
    SPREAD_TRAVEL_SECONDS,
    waveLook,
} from "./bonusClickEffects.ts"
import type {SpreadClick} from "../../backends/backend.ts"

// Tile 1 on the ground facing +Z, and six tiles round it. Tile 2 is due east of
// it, and the rest follow counter-clockwise.
const STEP = 0.004
const around = Array.from({length: 6}, (_, i) => {
    const angle = (i * Math.PI) / 3
    return new THREE.Vector3(Math.cos(angle) * STEP, Math.sin(angle) * STEP, 1).normalize()
})
const positions = new Float32Array([0, 0, 1, ...around.flatMap(p => [p.x, p.y, p.z])])
const tileAt = (tile: number) => new THREE.Vector3(
    positions[(tile - 1) * 3], positions[(tile - 1) * 3 + 1], positions[(tile - 1) * 3 + 2]).normalize()

// Out of order, the way the server's map hands them over.
const spread: SpreadClick = {countryId: "br", tile: 1, spread: [5, 2, 7, 3, 6, 4]}
const boosted = 1

describe("choreographSpread", () => {
    it("bursts at the tile clicked and throws one spark onto each tile around it", () => {
        const {sparks, centre} = choreographSpread(spread, positions)

        const bursts = sparks.filter(spark => spark.role === "burst")
        const landings = sparks.filter(spark => spark.role === "landing")

        expect(bursts).toHaveLength(1)
        expect(bursts[0].to.distanceTo(tileAt(1))).toBeLessThan(1e-6)
        expect(centre.distanceTo(tileAt(1))).toBeLessThan(1e-6)
        expect(landings).toHaveLength(6)
        for (const landing of landings) expect(landing.from.distanceTo(tileAt(1))).toBeLessThan(1e-6)
    })

    it("throws round the circle rather than in the order the server sent", () => {
        const landings = choreographSpread(spread, positions).sparks.filter(spark => spark.role === "landing")
        const order = landings.map(landing => [2, 3, 4, 5, 6, 7].find(tile => tileAt(tile).distanceTo(landing.to) < 1e-6))

        // Counter-clockwise, starting just past due west: 6 is at 240°, 5 at 180°.
        expect(order).toEqual([6, 7, 2, 3, 4, 5])
        landings.forEach((landing, i) => expect(landing.start).toBeCloseTo(SPREAD_THROW_FROM + i * SPREAD_THROW_STAGGER, 9))
    })

    it("lands every spark before the effect is over", () => {
        const last = choreographSpread(spread, positions).sparks.at(-1)!
        expect(last.start + last.travel).toBeLessThan(SPREAD_LIFETIME_SECONDS / 2)
    })

    it("still bursts on a lone island, which spreads onto nothing", () => {
        const {sparks, reach} = choreographSpread({...spread, spread: []}, positions)

        expect(sparks.map(spark => spark.role)).toEqual(["burst"])
        expect(reach).toBeGreaterThan(0)
    })
})

describe("choreographBoost", () => {
    it("flashes, shoots its streaks evenly apart, and runs three rings one after another", () => {
        const {sparks, waves, centre} = choreographBoost(boosted, positions)
        const streaks = sparks.filter(spark => spark.role === "streak")

        expect(sparks.filter(spark => spark.role === "burst")).toHaveLength(1)
        expect(streaks).toHaveLength(BOOST_STREAKS)
        expect(waves.map(wave => wave.startsAt)).toEqual(BOOST_WAVES_AT)

        const directions = streaks.map(streak => streak.to.clone().sub(centre).normalize())
        expect(directions[0].angleTo(directions[1])).toBeCloseTo((2 * Math.PI) / 3, 2)
        expect(directions[1].angleTo(directions[2])).toBeCloseTo((2 * Math.PI) / 3, 2)
    })

    it("turns the star by tile, the same way on every screen", () => {
        const one = choreographBoost(boosted, positions).sparks[1].to
        const again = choreographBoost(boosted, positions).sparks[1].to
        const other = choreographBoost(2, positions).sparks[1].to.clone()
            .sub(tileAt(2)).normalize()

        expect(one.distanceTo(again)).toBe(0)
        expect(one.clone().sub(tileAt(1)).normalize().angleTo(other)).toBeGreaterThan(0.1)
    })

    it("is over quicker than a spread: a boosted player clicks fast", () => {
        expect(BOOST_LIFETIME_SECONDS).toBeLessThan(SPREAD_LIFETIME_SECONDS)
    })
})

describe("sparkLook", () => {
    const origin = new THREE.Vector3(0, 0, 1)
    const burst: Spark = {from: origin, to: origin, start: 0, travel: 0, role: "burst"}
    const landing: Spark = {from: origin, to: origin, start: 0.1, travel: SPREAD_TRAVEL_SECONDS, role: "landing"}
    const streak: Spark = {from: origin, to: origin, start: 0, travel: BOOST_TRAVEL_SECONDS, role: "streak"}

    it("draws nothing before a spark's turn or after the effect", () => {
        expect(sparkLook(landing, 0.05, 1).glow).toBe(0)
        expect(sparkLook(burst, 1, 1).glow).toBe(0)
    })

    it("flashes a burst at once, then settles", () => {
        const lit = sparkLook(burst, 0, 1)
        const settled = sparkLook(burst, 0.5, 1)

        expect(lit.white).toBeGreaterThan(settled.white)
        expect(lit.scale).toBeGreaterThan(settled.scale)
    })

    it("flies a landing spark out, then pops it on its tile", () => {
        const flying = sparkLook(landing, 0.1 + SPREAD_TRAVEL_SECONDS / 2, 1)
        const landed = sparkLook(landing, 0.1 + SPREAD_TRAVEL_SECONDS, 1)
        const later = sparkLook(landing, 0.1 + SPREAD_TRAVEL_SECONDS + 0.4, 1)

        expect(flying.progress).toBeGreaterThan(0)
        expect(flying.progress).toBeLessThan(1)
        expect(landed.progress).toBe(1)
        expect(landed.scale).toBeGreaterThan(flying.scale)
        expect(later.scale).toBeLessThan(landed.scale)
        expect(later.glow).toBeGreaterThan(0)
    })

    it("is gone once a streak has run its course", () => {
        expect(sparkLook(streak, BOOST_TRAVEL_SECONDS * 0.2, 1).glow).toBeGreaterThan(0)
        expect(sparkLook(streak, BOOST_TRAVEL_SECONDS, 1).glow).toBe(0)
    })

    it("fades everything to nothing by the end", () => {
        expect(sparkLook(landing, 0.99, 1).glow).toBeLessThan(0.01)
    })

    it("keeps still for less motion: no flight, no streak, no flash", () => {
        expect(sparkLook(landing, 0.1, 1, true).progress).toBe(1)
        expect(sparkLook(landing, 0.1, 1, true).scale).toBe(sparkLook(landing, 0.6, 1, true).scale)
        expect(sparkLook(streak, 0.05, 1, true).glow).toBe(0)
        expect(sparkLook(burst, 0, 1, true).white).toBe(0)
    })
})

describe("waveLook", () => {
    it("runs only for its own stretch, outwards and fading", () => {
        const wave = {startsAt: 0.1, seconds: 0.5}

        expect(waveLook(wave, 0.09)).toBeUndefined()
        expect(waveLook(wave, 0.6)).toBeUndefined()

        const early = waveLook(wave, 0.15)!
        const late = waveLook(wave, 0.5)!
        expect(late.radius).toBeGreaterThan(early.radius)
        expect(late.opacity).toBeLessThan(early.opacity)
    })
})

describe("createBonusClickEffects", () => {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1)

    it("starts an effect on the next frame and takes it off once it is over", () => {
        const effects = createBonusClickEffects(positions)

        effects.playSpread(spread)
        effects.playBoost(boosted)
        expect(effects.object.children.length).toBeGreaterThan(0)

        effects.update(1000, camera, 800)
        effects.update(1000 + BOOST_LIFETIME_SECONDS, camera, 800)
        const spreadOnly = effects.object.children.length
        expect(spreadOnly).toBeGreaterThan(0)

        effects.update(1000 + SPREAD_LIFETIME_SECONDS, camera, 800)
        expect(effects.object.children).toHaveLength(0)

        effects.dispose()
    })

    it("keeps a fast run of clicks to a bounded number on screen", () => {
        const effects = createBonusClickEffects(positions)

        effects.playBoost(boosted)
        const perClick = effects.object.children.length
        for (let i = 0; i < 100; i++) effects.playBoost(boosted)

        expect(effects.object.children.length).toBeLessThan(perClick * 100)
        effects.dispose()
        expect(effects.object.children).toHaveLength(0)
    })
})
