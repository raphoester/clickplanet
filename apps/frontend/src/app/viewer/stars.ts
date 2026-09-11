import * as THREE from "three"

import starsVertex from "./shaders/stars/vertex.glsl"
import starsFragment from "./shaders/stars/fragment.glsl"

const STAR_COUNT = 9000

/**
 * Fixed, so the same sky comes back on every load rather than being reshuffled
 * each time the globe is mounted.
 */
const SEED = 0x9e3779b9

/** Anywhere between the near and far planes: only the direction is on screen. */
const SKY_RADIUS = 100

const SKY_FOV = 60

export type Sky = {
    /** Directions on a sphere of `SKY_RADIUS`, three floats per star. */
    positions: Float32Array
    /** Drawn diameter in CSS pixels, one float per star. */
    sizes: Float32Array
    /** Colour already scaled by the star's brightness, three floats per star. */
    tints: Float32Array
}

export type Starfield = {
    /**
     * Draws the sky, then `drawScene` over it, in one pass of the frame.
     *
     * The starfield owns the clearing for both: the caller must not clear
     * again, or it would wipe the sky it was just given.
     */
    render(renderer: THREE.WebGLRenderer, mainCamera: THREE.Camera, drawScene: () => void): void
    dispose(): void
}

/**
 * The night sky behind the globe.
 *
 * It is a **pass of its own**, with its own scene and its own camera, rather
 * than points added to the main scene, because the main camera is orthographic.
 * Its frustum is a box, so stars added to it would be clipped to a narrow tube
 * around the globe instead of covering the screen, and `camera.zoom` scales x
 * and y, so zooming in would fan the sky out across the screen as if it were
 * flying at the viewer.
 *
 * This camera copies the main camera's orientation and nothing else — no zoom,
 * no position — so the sky turns with the view and holds still through a zoom.
 */
export function createStarfield(): Starfield {
    const {positions, sizes, tints} = buildSky(STAR_COUNT, SEED)

    const geometry = new THREE.BufferGeometry()
    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
    geometry.setAttribute('size', new THREE.BufferAttribute(sizes, 1))
    geometry.setAttribute('tint', new THREE.BufferAttribute(tints, 3))

    const pixelRatio: THREE.IUniform = {value: 1}

    const material = new THREE.ShaderMaterial({
        uniforms: {pixelRatio},
        vertexShader: starsVertex,
        fragmentShader: starsFragment,
        transparent: true,
        depthTest: false,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
    })

    const points = new THREE.Points(geometry, material)
    // The camera sits at the centre of the sphere, so the sky is never off screen.
    points.frustumCulled = false

    const scene = new THREE.Scene()
    scene.add(points)

    const camera = new THREE.PerspectiveCamera(SKY_FOV, 1, 1, SKY_RADIUS * 2)

    return {
        render(renderer, mainCamera, drawScene) {
            const {width, height} = renderer.domElement
            const aspect = height > 0 ? width / height : 1
            if (camera.aspect !== aspect) {
                camera.aspect = aspect
                camera.updateProjectionMatrix()
            }

            camera.quaternion.copy(mainCamera.quaternion)
            pixelRatio.value = renderer.getPixelRatio()

            const previousAutoClear = renderer.autoClear
            renderer.autoClear = false
            try {
                renderer.clear()
                renderer.render(scene, camera)
                // The sky writes no depth, but the globe is only safe to draw
                // over it as long as that stays true.
                renderer.clearDepth()
                drawScene()
            } finally {
                // The picker renders to its own target and needs the clear back.
                renderer.autoClear = previousAutoClear
            }
        },
        dispose() {
            scene.clear()
            geometry.dispose()
            material.dispose()
        },
    }
}

/**
 * Stars spread evenly over a sphere, at sizes and brightnesses that vary enough
 * not to read as a grid.
 *
 * The even spread is the point: drawing a random latitude and a random
 * longitude packs stars around the poles, because the rings of longitude close
 * up there. Drawing the *height* uniformly instead gives equal area per band
 * (Archimedes), so no direction is favoured.
 */
export function buildSky(count: number, seed: number): Sky {
    const random = mulberry32(seed)

    const positions = new Float32Array(count * 3)
    const sizes = new Float32Array(count)
    const tints = new Float32Array(count * 3)

    for (let i = 0; i < count; i++) {
        const height = 1 - 2 * random()
        const ring = Math.sqrt(Math.max(0, 1 - height * height))
        const angle = 2 * Math.PI * random()

        positions[i * 3] = ring * Math.cos(angle) * SKY_RADIUS
        positions[i * 3 + 1] = height * SKY_RADIUS
        positions[i * 3 + 2] = ring * Math.sin(angle) * SKY_RADIUS

        // Both curves are weighted low: a sky of mostly faint pinpricks with a
        // scattering of brighter ones, rather than a field competing with the
        // flags in front of it.
        sizes[i] = 1.1 + 1.9 * Math.pow(random(), 2.5)
        const brightness = 0.22 + 0.58 * Math.pow(random(), 2.2)

        // A touch of warm or cool, far short of a colour anyone would name.
        const temperature = 2 * random() - 1
        tints[i * 3] = brightness * (1 + 0.09 * temperature)
        tints[i * 3 + 1] = brightness
        tints[i * 3 + 2] = brightness * (1 - 0.09 * temperature)
    }

    return {positions, sizes, tints}
}

/** Mulberry32: a small, well-spread PRNG, here only so the sky is repeatable. */
export function mulberry32(seed: number): () => number {
    let state = seed >>> 0

    return () => {
        state = (state + 0x6d2b79f5) >>> 0
        let t = state
        t = Math.imul(t ^ (t >>> 15), t | 1)
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296
    }
}
