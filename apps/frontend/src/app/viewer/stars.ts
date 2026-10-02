import * as THREE from "three"

import starsVertex from "./shaders/stars/vertex.glsl"
import starsFragment from "./shaders/stars/fragment.glsl"

const STAR_COUNT = 9000

const SEED = 0x9e3779b9

const SKY_RADIUS = 100

const SKY_FOV = 60

export type Sky = {
    positions: Float32Array
    sizes: Float32Array
    tints: Float32Array
}

export type Starfield = {
    // Clears for both passes: the caller must not clear again, or it wipes the sky.
    render(renderer: THREE.WebGLRenderer, mainCamera: THREE.Camera, drawScene: () => void): void
    dispose(): void
}

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
                renderer.clearDepth()
                drawScene()
            } finally {
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

        sizes[i] = 1.1 + 1.9 * Math.pow(random(), 2.5)
        const brightness = 0.22 + 0.58 * Math.pow(random(), 2.2)

        const temperature = 2 * random() - 1
        tints[i * 3] = brightness * (1 + 0.09 * temperature)
        tints[i * 3 + 1] = brightness
        tints[i * 3 + 2] = brightness * (1 - 0.09 * temperature)
    }

    return {positions, sizes, tints}
}

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
