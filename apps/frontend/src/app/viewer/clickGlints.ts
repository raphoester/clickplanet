import * as THREE from "three"
import {tilePointSize} from "./pointSize.ts"

import glintVertex from "./shaders/clickGlint/vertex.glsl"
import glintFragment from "./shaders/clickGlint/fragment.glsl"

const LIFT = 1.003

const MAX_PLAYING = 64

export const GLINT_SECONDS = 0.7
const ATTACK_SECONDS = 0.06
export const PEAK = 0.9
const WHITE = 0.35

export const TILES_WIDE = 1.8
export const MIN_GLINT_PX = 9

const SKY = new THREE.Color(0.35, 0.75, 1.0)

export type GlintLook = {
    opacity: number
    white: number
}

export function glintLook(age: number): GlintLook | undefined {
    if (age < 0 || age >= GLINT_SECONDS) return undefined

    const rise = smoothstep(0, ATTACK_SECONDS, age)
    const left = 1 - Math.max(0, age - ATTACK_SECONDS) / (GLINT_SECONDS - ATTACK_SECONDS)

    return {opacity: PEAK * rise * left * left, white: WHITE * left * left}
}

export function glintSize(tilePx: number, pixelRatio: number): number {
    return Math.max(tilePx * TILES_WIDE, MIN_GLINT_PX * pixelRatio)
}

export function inView(point: THREE.Vector3, camera: THREE.Camera): boolean {
    if (point.dot(camera.getWorldDirection(new THREE.Vector3())) >= 0) return false

    const {x, y} = point.clone().project(camera)
    return Math.abs(x) <= 1 && Math.abs(y) <= 1
}

function smoothstep(from: number, to: number, x: number): number {
    const t = Math.min(1, Math.max(0, (x - from) / (to - from)))
    return t * t * (3 - 2 * t)
}

export type ClickGlints = {
    readonly object: THREE.Object3D
    setOwnHue(hue: number | undefined): void
    playOwnClick(tile: number, camera: THREE.Camera): void
    playClick(tile: number, camera: THREE.Camera): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number): boolean
    dispose(): void
}

type Playing = {
    points: THREE.Points
    material: THREE.ShaderMaterial
    startedAt: number | undefined
}

export function createClickGlints(positions: ArrayLike<number>): ClickGlints {
    const group = new THREE.Group()
    group.renderOrder = 1

    const spot = new THREE.BufferGeometry()
    spot.setAttribute("position", new THREE.BufferAttribute(new Float32Array(3), 3))

    let playing: Playing[] = []
    let ownColour: THREE.Color | undefined

    const centreOf = (tile: number) => new THREE.Vector3(
        positions[(tile - 1) * 3], positions[(tile - 1) * 3 + 1], positions[(tile - 1) * 3 + 2]).normalize()

    const stop = (glint: Playing) => {
        group.remove(glint.points)
        glint.material.dispose()
    }

    const play = (centre: THREE.Vector3, colour: THREE.Color) => {
        if (typeof document !== "undefined" && document.visibilityState === "hidden") return

        const material = new THREE.ShaderMaterial({
            uniforms: {
                size: {value: 0},
                unitsPerPixel: {value: 0},
                colour: {value: colour},
                opacity: {value: 0},
                white: {value: 0},
            },
            vertexShader: glintVertex,
            fragmentShader: glintFragment,
            transparent: true,
            depthWrite: false,
        })

        const points = new THREE.Points(spot, material)
        points.position.copy(centre).multiplyScalar(LIFT)
        points.frustumCulled = false
        points.renderOrder = 1
        group.add(points)

        playing.push({points, material, startedAt: undefined})

        while (playing.length > MAX_PLAYING) {
            const oldest = playing.shift()
            if (oldest) stop(oldest)
        }
    }

    const playInView = (tile: number, camera: THREE.Camera, colour: THREE.Color) => {
        const centre = centreOf(tile)
        if (inView(centre, camera)) play(centre, colour)
    }

    const update = (seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number) => {
        if (playing.length === 0) return false

        const tilePx = tilePointSize(camera.zoom, viewportHeight)
        const unitsPerPixel = 1 / ((viewportHeight / 2) * camera.zoom)

        playing = playing.filter((glint) => {
            glint.startedAt ??= seconds
            const look = glintLook(seconds - glint.startedAt)
            if (!look) {
                stop(glint)
                return false
            }

            const {uniforms} = glint.material
            uniforms.size.value = glintSize(tilePx, pixelRatio)
            uniforms.unitsPerPixel.value = unitsPerPixel
            uniforms.opacity.value = look.opacity
            uniforms.white.value = look.white

            return true
        })

        return true
    }

    return {
        object: group,
        setOwnHue: (hue) => {
            ownColour = hue === undefined ? undefined : new THREE.Color().setHSL(hue / 360, 0.85, 0.62)
        },
        playOwnClick: (tile, camera) => playInView(tile, camera, ownColour ?? SKY),
        playClick: (tile, camera) => playInView(tile, camera, SKY),
        update,
        dispose: () => {
            for (const glint of playing) stop(glint)
            playing = []
            spot.dispose()
        },
    }
}
