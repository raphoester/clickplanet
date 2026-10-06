import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    createClickGlints,
    GLINT_SECONDS,
    glintLook,
    glintSize,
    inView,
    MIN_GLINT_PX,
    PEAK,
    TILES_WIDE,
} from "./clickGlints.ts"

const facing = new Float32Array([0, 0, 1])

const lookingAtFront = () => {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 10)
    camera.position.set(0, 0, 5)
    camera.lookAt(0, 0, 0)
    camera.updateMatrixWorld()
    return camera
}

const colourOf = (object: THREE.Object3D) =>
    ((object as THREE.Points).material as THREE.ShaderMaterial).uniforms.colour.value as THREE.Color

describe("glintLook", () => {
    it("lights at once and is gone well inside a second", () => {
        expect(glintLook(0.06, "click")!.opacity).toBeCloseTo(PEAK)
        expect(GLINT_SECONDS).toBeLessThan(1)
        expect(glintLook(GLINT_SECONDS, "click")).toBeUndefined()
        expect(glintLook(GLINT_SECONDS, "hit")).toBeUndefined()
        expect(glintLook(GLINT_SECONDS, "reinforced")).toBeUndefined()
    })

    it("fades a take where it stands", () => {
        const early = glintLook(0.1, "click")!
        const late = glintLook(0.6, "click")!

        expect(late.opacity).toBeLessThan(early.opacity)
        expect(early.scale).toBe(1)
        expect(late.scale).toBe(1)
    })

    it("shrinks a hit as it fades", () => {
        const early = glintLook(0.1, "hit")!
        const late = glintLook(0.6, "hit")!

        expect(late.scale).toBeLessThan(early.scale)
        expect(late.scale).toBeGreaterThan(0)
        expect(late.opacity).toBeLessThan(early.opacity)
    })

    it("grows a reinforcement as it fades", () => {
        const early = glintLook(0.1, "reinforced")!
        const late = glintLook(0.6, "reinforced")!

        expect(late.scale).toBeGreaterThan(early.scale)
        expect(late.opacity).toBeLessThan(early.opacity)
    })
})

describe("glintSize", () => {
    it("never goes under its floor, so a click is seen from orbit", () => {
        expect(glintSize(1.5, 1, 1)).toBe(MIN_GLINT_PX)
        expect(glintSize(3, 2, 1)).toBe(MIN_GLINT_PX * 2)
    })

    it("sits on its tile up close", () => {
        expect(glintSize(60, 1, 1)).toBeCloseTo(60 * TILES_WIDE)
    })
})

describe("inView", () => {
    it("sees the side of the globe facing the camera", () => {
        expect(inView(new THREE.Vector3(0, 0, 1), lookingAtFront())).toBe(true)
        expect(inView(new THREE.Vector3(0.6, 0, 0.8), lookingAtFront())).toBe(true)
    })

    it("does not see the far side, though it projects inside the screen", () => {
        expect(inView(new THREE.Vector3(0, 0, -1), lookingAtFront())).toBe(false)
    })

    it("does not see what a zoom pushed off the screen", () => {
        const zoomed = lookingAtFront()
        zoomed.zoom = 4
        zoomed.updateProjectionMatrix()

        expect(inView(new THREE.Vector3(0, 0, 1), zoomed)).toBe(true)
        expect(inView(new THREE.Vector3(0.6, 0, 0.8), zoomed)).toBe(false)
    })
})

describe("createClickGlints", () => {
    it("plays a click in view and takes it off once it is over", () => {
        const glints = createClickGlints(facing)
        const camera = lookingAtFront()

        glints.playClick(1, camera)
        expect(glints.object.children).toHaveLength(1)

        expect(glints.update(1000, camera, 800, 1)).toBe(true)
        expect(glints.update(1000 + GLINT_SECONDS, camera, 800, 1)).toBe(true)
        expect(glints.object.children).toHaveLength(0)
        expect(glints.update(1000 + GLINT_SECONDS + 0.1, camera, 800, 1)).toBe(false)

        glints.dispose()
    })

    it("plays nothing for a click nobody can see", () => {
        const glints = createClickGlints(new Float32Array([0, 0, -1]))
        const camera = lookingAtFront()

        glints.playClick(1, camera)
        glints.playOwnClick(1, camera)

        expect(glints.object.children).toHaveLength(0)
        expect(glints.update(1000, camera, 800, 1)).toBe(false)
        glints.dispose()
    })

    it("shows this player's clicks in its hue, and everyone else's in another colour", () => {
        const glints = createClickGlints(facing)
        const camera = lookingAtFront()

        glints.setOwnHue(270)
        glints.playOwnClick(1, camera)
        glints.playClick(1, camera)

        const [click, theirs] = glints.object.children.map(colourOf)
        expect(click.getHSL({h: 0, s: 0, l: 0}).h * 360).toBeCloseTo(270)
        expect(theirs.equals(click)).toBe(false)

        glints.dispose()
    })

    it("tells a hit and a reinforcement from a click, and from each other, by colour", () => {
        const glints = createClickGlints(facing)
        const camera = lookingAtFront()

        glints.playClick(1, camera)
        glints.playHit(1, camera)
        glints.playReinforced(1, camera)

        const colours = glints.object.children.map(colourOf)
        expect(new Set(colours.map((colour) => colour.getHexString())).size).toBe(3)

        glints.dispose()
    })

    it("plays no hit and no reinforcement nobody can see", () => {
        const glints = createClickGlints(new Float32Array([0, 0, -1]))
        const camera = lookingAtFront()

        glints.playHit(1, camera)
        glints.playReinforced(1, camera)

        expect(glints.object.children).toHaveLength(0)
        glints.dispose()
    })

    it("keeps a fast run of clicks to a bounded number on screen", () => {
        const glints = createClickGlints(facing)
        const camera = lookingAtFront()

        for (let i = 0; i < 200; i++) glints.playClick(1, camera)

        expect(glints.object.children.length).toBeLessThan(200)
        glints.dispose()
        expect(glints.object.children).toHaveLength(0)
    })
})
