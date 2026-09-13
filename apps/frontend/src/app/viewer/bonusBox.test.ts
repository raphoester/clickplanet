import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    BOX_PALETTE,
    boxScale,
    burstAt,
    FACE_COLOURS,
    FACE_TONES,
    haloColourAt,
    flightOpacity,
    isBehindGlobe,
    Orbit,
    orbitFromSeed,
    orbitPosition,
} from "./bonusBox.ts"

const ORBIT_RADIUS = 1.15

describe("orbitFromSeed", () => {
    it("gives the same orbit for the same seed, so every client draws one path", () => {
        const first = orbitFromSeed(7)
        const second = orbitFromSeed(7)

        expect(first.u.toArray()).toEqual(second.u.toArray())
        expect(first.v.toArray()).toEqual(second.v.toArray())
        expect(first.phase).toEqual(second.phase)
    })

    it("gives a different orbit for a different seed", () => {
        expect(orbitFromSeed(1).u.toArray()).not.toEqual(orbitFromSeed(2).u.toArray())
    })

    it("spans the plane with two perpendicular unit vectors", () => {
        for (let seed = 0; seed < 200; seed++) {
            const {u, v} = orbitFromSeed(seed)

            expect(u.length()).toBeCloseTo(1, 6)
            expect(v.length()).toBeCloseTo(1, 6)
            expect(u.dot(v)).toBeCloseTo(0, 6)
        }
    })

    it("tilts the orbits across the whole sphere rather than favouring one plane", () => {
        // The plane's own axis is u × v. Over many seeds its height should cover
        // the range, not cluster: a drawn latitude would pack them near one band.
        const heights = Array.from({length: 400}, (_, seed) => {
            const {u, v} = orbitFromSeed(seed)
            return new THREE.Vector3().crossVectors(u, v).y
        })

        expect(Math.min(...heights)).toBeLessThan(-0.8)
        expect(Math.max(...heights)).toBeGreaterThan(0.8)
    })
})

describe("orbitPosition", () => {
    it("keeps the box clear of the tile shell all the way round", () => {
        const orbit = orbitFromSeed(3)

        for (let step = 0; step < 64; step++) {
            const position = orbitPosition(orbit, (step / 64) * 2 * Math.PI)

            expect(position.length()).toBeCloseTo(ORBIT_RADIUS, 6)
            expect(position.length()).toBeGreaterThan(1)
        }
    })

    it("stays in its own plane", () => {
        const orbit = orbitFromSeed(11)
        const axis = new THREE.Vector3().crossVectors(orbit.u, orbit.v)

        for (let step = 0; step < 32; step++) {
            expect(orbitPosition(orbit, step).dot(axis)).toBeCloseTo(0, 6)
        }
    })

    it("comes back round after a full turn", () => {
        const orbit = orbitFromSeed(5)

        const start = orbitPosition(orbit, 0)
        const round = orbitPosition(orbit, 2 * Math.PI)

        expect(round.distanceTo(start)).toBeCloseTo(0, 6)
    })

    it("starts where the seed's phase puts it, not at a fixed point", () => {
        const first = orbitPosition(orbitFromSeed(1), 0)
        const second = orbitPosition(orbitFromSeed(2), 0)

        expect(first.distanceTo(second)).toBeGreaterThan(0.01)
    })
})

describe("isBehindGlobe", () => {
    // The camera looks down -Z, so it is this way from the scene.
    const toCamera = new THREE.Vector3(0, 0, 1)

    it("sees a box on the near side", () => {
        expect(isBehindGlobe(new THREE.Vector3(0, 0, ORBIT_RADIUS), toCamera)).toBe(false)
    })

    it("hides one directly behind the planet", () => {
        expect(isBehindGlobe(new THREE.Vector3(0, 0, -ORBIT_RADIUS), toCamera)).toBe(true)
    })

    it("sees one past the limb even though it is further away than the centre", () => {
        // Beyond the globe's radius from the view axis, so nothing is in front of
        // it: this is the case a plain depth comparison would get wrong.
        expect(isBehindGlobe(new THREE.Vector3(1.1, 0, -0.3), toCamera)).toBe(false)
    })

    it("hides one just inside the limb", () => {
        expect(isBehindGlobe(new THREE.Vector3(0.95, 0, -0.6), toCamera)).toBe(true)
    })

    it("follows the camera round rather than assuming one direction", () => {
        const fromTheSide = new THREE.Vector3(1, 0, 0)
        const position = new THREE.Vector3(-ORBIT_RADIUS, 0, 0)

        expect(isBehindGlobe(position, fromTheSide)).toBe(true)
        expect(isBehindGlobe(position, fromTheSide.clone().negate())).toBe(false)
    })

    it("leaves an edge-on orbit on view for the two thirds of it that geometry allows", () => {
        // Worst case: the plane holds the view axis, so the box goes right behind
        // the centre. Hidden only while it is both past the centre and inside the
        // limb, which is an arc of 2 * acos(1 / 1.15) short of the full half.
        const hidden = (Math.PI - 2 * Math.acos(1 / ORBIT_RADIUS)) / (2 * Math.PI)

        const edgeOn = {u: new THREE.Vector3(1, 0, 0), v: new THREE.Vector3(0, 0, 1), phase: 0}

        expect(visibleShare(edgeOn)).toBeCloseTo(1 - hidden, 2)
    })

    it("never hides a box for more of its turn than that, whatever the tilt", () => {
        const floor = 1 - (Math.PI - 2 * Math.acos(1 / ORBIT_RADIUS)) / (2 * Math.PI)

        for (let seed = 0; seed < 100; seed++) {
            expect(visibleShare(orbitFromSeed(seed))).toBeGreaterThanOrEqual(floor - 0.01)
        }
    })

    function visibleShare(orbit: Orbit): number {
        const steps = 3600

        let visible = 0
        for (let step = 0; step < steps; step++) {
            const position = orbitPosition(orbit, (step / steps) * 2 * Math.PI)
            if (!isBehindGlobe(position, toCamera)) visible++
        }

        return visible / steps
    }
})

describe("FACE_TONES", () => {
    it("paints one key light: a bright top, mid sides and a dark underside", () => {
        const [plusX, minusX, plusY, minusY, plusZ, minusZ] = FACE_TONES

        expect(plusY).toBeGreaterThan(plusX)
        expect(minusY).toBeLessThan(plusX)
        expect([minusX, plusZ, minusZ]).toEqual([plusX, plusX, plusX])
    })

    it("covers all six faces of the box, in BoxGeometry's material order", () => {
        expect(FACE_TONES).toHaveLength(6)
        expect(new THREE.BoxGeometry(1, 1, 1).groups).toHaveLength(FACE_TONES.length)
    })

    it("keeps every face inside the texture's own brightness", () => {
        for (const tone of FACE_TONES) {
            expect(tone).toBeGreaterThan(0)
            expect(tone).toBeLessThanOrEqual(1)
        }
    })
})

describe("boxScale", () => {
    it("keeps a constant world size at the zooms the box is actually flown at", () => {
        // The complaint this answers: dividing the zoom out held the box at a
        // fixed pixel size while the ground under it grew, which reads as the
        // box being stuck to the screen rather than flying over the planet.
        expect(boxScale(1)).toBe(boxScale(5))
        expect(boxScale(1)).toBe(boxScale(9))
    })

    it("caps it before it could take over the screen", () => {
        // The visible world is 2/zoom tall, so an uncapped box would be most of
        // the frame here — a zoom the orbit is barely ever in view at anyway.
        expect(boxScale(50)).toBeLessThan(boxScale(1))
        expect(boxScale(50) * 50).toBeLessThanOrEqual(1)
    })

    it("never grows as the player zooms in", () => {
        let previous = boxScale(1)
        for (let zoom = 1; zoom <= 50; zoom++) {
            const scale = boxScale(zoom)
            expect(scale).toBeLessThanOrEqual(previous)
            expect(scale).toBeGreaterThan(0)
            previous = scale
        }
    })
})

describe("flightOpacity", () => {
    it("holds the box solid for most of its life", () => {
        expect(flightOpacity(0, 7, 0.9)).toBe(1)
        expect(flightOpacity(5, 7, 0.9)).toBe(1)
    })

    it("fades it out rather than cutting it off", () => {
        expect(flightOpacity(6.55, 7, 0.9)).toBeCloseTo(0.5, 2)
        expect(flightOpacity(6.9, 7, 0.9)).toBeLessThan(0.2)
    })

    it("is gone once its life is up, and stays gone", () => {
        expect(flightOpacity(7, 7, 0.9)).toBe(0)
        expect(flightOpacity(90, 7, 0.9)).toBe(0)
    })

    it("only ever falls", () => {
        let previous = flightOpacity(0, 7, 0.9)
        for (let step = 0; step <= 100; step++) {
            const opacity = flightOpacity((step / 100) * 8, 7, 0.9)
            expect(opacity).toBeLessThanOrEqual(previous)
            previous = opacity
        }
    })
})

describe("burstAt", () => {
    it("starts where the box already was, so the pop does not jump", () => {
        expect(burstAt(0, 0.45)).toEqual({scale: 1, opacity: 1})
    })

    it("swells and fades out together", () => {
        const half = burstAt(0.225, 0.45)

        expect(half.scale).toBeGreaterThan(1)
        expect(half.opacity).toBeCloseTo(0.5, 2)
    })

    it("eases out, so most of the growth is in the first half", () => {
        const half = burstAt(0.225, 0.45)
        const end = burstAt(0.45, 0.45)

        expect(half.scale - 1).toBeGreaterThan((end.scale - 1) / 2)
    })

    it("ends invisible, and clamps past the end rather than inverting", () => {
        expect(burstAt(0.45, 0.45).opacity).toBe(0)
        expect(burstAt(10, 0.45).opacity).toBe(0)
        expect(burstAt(10, 0.45).scale).toEqual(burstAt(0.45, 0.45).scale)
    })

    it("clamps a negative age instead of shrinking the box", () => {
        expect(burstAt(-1, 0.45)).toEqual({scale: 1, opacity: 1})
    })
})

describe("the box's colours", () => {
    // BoxGeometry's material order pairs the faces: +X/-X, +Y/-Y, +Z/-Z.
    const OPPOSITE = [1, 0, 3, 2, 5, 4]

    it("colours all six faces, and wears every colour in the palette", () => {
        expect(FACE_COLOURS).toHaveLength(6)
        expect(new Set(FACE_COLOURS)).toEqual(new Set(Object.keys(BOX_PALETTE)))
    })

    it("never shows one colour on two faces at once", () => {
        // A cube shows at most one face of each opposite pair, so two faces that
        // are not opposite can be on screen together.
        for (let face = 0; face < 6; face++) {
            for (let other = 0; other < 6; other++) {
                if (other === face || other === OPPOSITE[face]) continue
                expect(FACE_COLOURS[face]).not.toBe(FACE_COLOURS[other])
            }
        }
    })
})

describe("haloColourAt", () => {
    const glowOf = (colour: keyof typeof BOX_PALETTE) => new THREE.Color(...BOX_PALETTE[colour].glow)

    it("starts on the classic gold", () => {
        expect(haloColourAt(0).equals(glowOf("gold"))).toBe(true)
    })

    it("passes through every colour of the palette", () => {
        const seen = new Set<string>()
        for (let step = 0; step < 4; step++) {
            const colour = haloColourAt(step * 0.7)
            for (const name of Object.keys(BOX_PALETTE) as (keyof typeof BOX_PALETTE)[]) {
                if (colour.equals(glowOf(name))) seen.add(name)
            }
        }

        expect(seen.size).toBe(4)
    })

    it("comes back round to where it started", () => {
        expect(haloColourAt(4 * 0.7).getHex()).toBe(haloColourAt(0).getHex())
    })

    it("eases between colours rather than jumping", () => {
        for (let frame = 0; frame < 300; frame++) {
            const now = haloColourAt(frame / 60)
            const next = haloColourAt((frame + 1) / 60)

            expect(Math.abs(now.r - next.r) + Math.abs(now.g - next.g) + Math.abs(now.b - next.b)).toBeLessThan(0.1)
        }
    })

    it("writes into the colour it is given, so a frame allocates nothing", () => {
        const into = new THREE.Color()

        expect(haloColourAt(1, into)).toBe(into)
    })
})
