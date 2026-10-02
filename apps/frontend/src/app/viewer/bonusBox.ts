import * as THREE from "three"

import glowVertex from "./shaders/atmosphere/vertex.glsl"
import glowFragment from "./shaders/atmosphere/fragment.glsl"

import {mulberry32} from "./stars.ts"
import {drawQuestionMark} from "./questionMark.ts"

const ORBIT_RADIUS = 1.15

const WORLD_SIZE = 0.1

const MAX_APPARENT_SIZE = 0.9

const SPIN_PER_SECOND = 0.8

const ORBIT_PER_SECOND = 0.4

const LIFETIME_SECONDS = 12

const FADE_SECONDS = 0.9

const BURST_SECONDS = 0.45
const BURST_SCALE = 2.2
const BURST_GLOW = 3

const GLOW_RADIUS = 1.25
const GLOW_POWER = 2.6
const GLOW_INTENSITY = 1.4

export const BOX_PALETTE = {
    gold: {face: "#f2a91c", edge: "#4a2c02", glow: [1.0, 0.68, 0.16]},
    red: {face: "#e0412f", edge: "#4a0e07", glow: [1.0, 0.36, 0.3]},
    blue: {face: "#2f8cff", edge: "#0a2a55", glow: [0.35, 0.65, 1.0]},
    green: {face: "#25b872", edge: "#0b3d25", glow: [0.35, 0.95, 0.6]},
} as const

export type BoxColour = keyof typeof BOX_PALETTE

export const FACE_COLOURS: readonly BoxColour[] = ["blue", "green", "gold", "gold", "red", "red"]

const HALO_CYCLE: readonly BoxColour[] = ["gold", "red", "blue", "green"]
const HALO_STEP_SECONDS = 0.7

export const FACE_TONES = [0.74, 0.74, 1, 0.4, 0.74, 0.74]

const TEXTURE_SIZE = 128

export type Orbit = {
    u: THREE.Vector3
    v: THREE.Vector3
    phase: number
}

export type BonusBox = {
    readonly object: THREE.Object3D

    readonly visible: boolean

    readonly flying: boolean

    spawn(seed: number, deadline?: number): void

    take(): boolean

    hide(): void

    update(seconds: number, camera: THREE.OrthographicCamera): boolean

    hitTest(camera: THREE.OrthographicCamera, ndc: THREE.Vector2): boolean

    dispose(): void
}

export function createBonusBox(): BonusBox {
    const textures = new Map<BoxColour, THREE.CanvasTexture>()
    const textureFor = (colour: BoxColour) => {
        let texture = textures.get(colour)
        if (!texture) {
            texture = faceTexture(colour)
            textures.set(colour, texture)
        }
        return texture
    }

    const geometry = new THREE.BoxGeometry(1, 1, 1)
    const materials = FACE_TONES.map((tone, face) => new THREE.MeshBasicMaterial({
        map: textureFor(FACE_COLOURS[face]),
        color: new THREE.Color(tone, tone, tone),
        transparent: true,
    }))

    const box = new THREE.Mesh(geometry, materials)

    const glowGeometry = new THREE.IcosahedronGeometry(GLOW_RADIUS, 8)
    const glowMaterial = new THREE.ShaderMaterial({
        uniforms: {
            colour: {value: haloColourAt(0)},
            power: {value: GLOW_POWER},
            intensity: {value: GLOW_INTENSITY},
        },
        vertexShader: glowVertex,
        fragmentShader: glowFragment,
        side: THREE.BackSide,
        transparent: true,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
    })

    const glow = new THREE.Mesh(glowGeometry, glowMaterial)

    // Box before halo: both are transparent with one centre, so three's sort could flip them.
    box.renderOrder = 0
    glow.renderOrder = 1

    const group = new THREE.Group()
    group.add(box)
    group.add(glow)
    group.visible = false

    const raycaster = new THREE.Raycaster()
    const toCamera = new THREE.Vector3()

    let orbit = orbitFromSeed(0)
    let phase: "gone" | "flying" | "taken" = "gone"
    let spawnedAt: number | undefined
    let takenAt: number | undefined
    let deadline = Infinity
    let lifetime = LIFETIME_SECONDS

    const setFade = (opacity: number, flare: number) => {
        for (const material of materials) material.opacity = opacity
        glowMaterial.uniforms.intensity.value = GLOW_INTENSITY * flare * opacity
    }

    const stop = () => {
        phase = "gone"
        group.visible = false
    }

    return {
        object: group,

        get visible() {
            return group.visible
        },

        get flying() {
            return phase === "flying"
        },

        spawn(seed: number, until = Infinity) {
            orbit = orbitFromSeed(seed)
            deadline = until
            spawnedAt = undefined
            takenAt = undefined
            phase = "flying"
            group.visible = true
            setFade(1, 1)
        },

        take() {
            if (phase !== "flying") return false

            phase = "taken"
            takenAt = undefined

            return true
        },

        hide: stop,

        update(seconds: number, camera: THREE.OrthographicCamera) {
            if (phase === "gone") return false

            if (spawnedAt === undefined) {
                spawnedAt = seconds
                lifetime = Math.min(LIFETIME_SECONDS, deadline - seconds)
            }

            const scale = boxScale(camera.zoom)
            haloColourAt(seconds, glowMaterial.uniforms.colour.value)

            if (phase === "taken") {
                if (takenAt === undefined) takenAt = seconds

                const pop = burstAt(seconds - takenAt)
                group.scale.setScalar(scale * pop.scale)
                setFade(pop.opacity, BURST_GLOW)

                if (pop.opacity <= 0) stop()
                return true
            }

            const age = seconds - spawnedAt

            group.position.copy(orbitPosition(orbit, age * ORBIT_PER_SECOND))
            group.scale.setScalar(scale)
            box.rotation.set(age * SPIN_PER_SECOND, age * SPIN_PER_SECOND * 0.7, 0)

            const opacity = flightOpacity(age, lifetime)
            setFade(opacity, 1)

            if (opacity <= 0) stop()

            return true
        },

        hitTest(camera: THREE.OrthographicCamera, ndc: THREE.Vector2) {
            if (phase !== "flying") return false

            camera.getWorldDirection(toCamera).negate()
            if (isBehindGlobe(group.position, toCamera)) return false

            group.updateMatrixWorld()
            raycaster.setFromCamera(ndc, camera)

            return raycaster.intersectObject(box, false).length > 0
        },

        dispose() {
            group.clear()
            geometry.dispose()
            for (const material of materials) material.dispose()
            glowGeometry.dispose()
            glowMaterial.dispose()
            for (const texture of textures.values()) texture.dispose()
        },
    }
}

export function boxScale(zoom: number): number {
    return Math.min(WORLD_SIZE, MAX_APPARENT_SIZE / zoom)
}

export function haloColourAt(seconds: number, into = new THREE.Color()): THREE.Color {
    const steps = seconds / HALO_STEP_SECONDS
    const index = Math.floor(steps)
    const through = steps - index
    const eased = through * through * (3 - 2 * through)

    const glowAt = (step: number) => {
        const length = HALO_CYCLE.length
        return BOX_PALETTE[HALO_CYCLE[((step % length) + length) % length]].glow
    }
    const from = glowAt(index)
    const to = glowAt(index + 1)

    return into.setRGB(
        from[0] + (to[0] - from[0]) * eased,
        from[1] + (to[1] - from[1]) * eased,
        from[2] + (to[2] - from[2]) * eased,
    )
}

export function flightOpacity(
    age: number,
    lifetime = LIFETIME_SECONDS,
    fade = FADE_SECONDS,
): number {
    const left = lifetime - age
    if (left <= 0) return 0

    return Math.min(1, left / fade)
}

export function burstAt(since: number, duration = BURST_SECONDS): {scale: number, opacity: number} {
    const through = Math.min(1, Math.max(0, since / duration))

    return {
        scale: 1 + (BURST_SCALE - 1) * (1 - (1 - through) * (1 - through)),
        opacity: 1 - through,
    }
}

export function orbitFromSeed(seed: number): Orbit {
    const random = mulberry32(seed)

    const height = 1 - 2 * random()
    const ring = Math.sqrt(Math.max(0, 1 - height * height))
    const angle = 2 * Math.PI * random()

    const axis = new THREE.Vector3(
        ring * Math.cos(angle),
        height,
        ring * Math.sin(angle),
    )

    const off = Math.abs(axis.y) > 0.9
        ? new THREE.Vector3(1, 0, 0)
        : new THREE.Vector3(0, 1, 0)

    const u = new THREE.Vector3().crossVectors(axis, off).normalize()
    const v = new THREE.Vector3().crossVectors(axis, u).normalize()

    return {u, v, phase: 2 * Math.PI * random()}
}

export function orbitPosition(orbit: Orbit, angle: number, radius = ORBIT_RADIUS): THREE.Vector3 {
    const theta = orbit.phase + angle

    return new THREE.Vector3()
        .addScaledVector(orbit.u, Math.cos(theta) * radius)
        .addScaledVector(orbit.v, Math.sin(theta) * radius)
}

export function isBehindGlobe(
    position: THREE.Vector3,
    toCamera: THREE.Vector3,
    globeRadius = 1,
): boolean {
    const depth = position.dot(toCamera)
    if (depth >= 0) return false

    const across = Math.sqrt(Math.max(0, position.lengthSq() - depth * depth))

    return across < globeRadius
}

function faceTexture(colour: BoxColour): THREE.CanvasTexture {
    const {face, edge} = BOX_PALETTE[colour]

    const canvas = document.createElement("canvas")
    canvas.width = TEXTURE_SIZE
    canvas.height = TEXTURE_SIZE

    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the bonus box face")

    const size = TEXTURE_SIZE

    const ground = context.createLinearGradient(0, 0, size, size)
    ground.addColorStop(0, mix(face, "#ffffff", 0.35))
    ground.addColorStop(0.55, face)
    ground.addColorStop(1, mix(face, edge, 0.3))
    context.fillStyle = ground
    context.fillRect(0, 0, size, size)

    context.fillStyle = "rgba(255, 255, 255, 0.22)"
    context.beginPath()
    context.moveTo(0, size * 0.18)
    context.lineTo(size * 0.18, 0)
    context.lineTo(size * 0.46, 0)
    context.lineTo(0, size * 0.46)
    context.closePath()
    context.fill()

    context.strokeStyle = edge
    context.lineWidth = size * 0.09
    context.strokeRect(context.lineWidth / 2, context.lineWidth / 2, size - context.lineWidth, size - context.lineWidth)

    context.fillStyle = edge
    const inset = size * 0.19
    const rivet = size * 0.035
    for (const [x, y] of [[inset, inset], [size - inset, inset], [inset, size - inset], [size - inset, size - inset]]) {
        context.beginPath()
        context.arc(x, y, rivet, 0, 2 * Math.PI)
        context.fill()
    }

    const mark = size * 0.56
    drawQuestionMark(context, (size - mark) / 2, (size - mark) / 2, mark, "#fff6df", edge)

    const texture = new THREE.CanvasTexture(canvas)
    texture.colorSpace = THREE.SRGBColorSpace

    return texture
}

function mix(a: string, b: string, amount: number): string {
    const channel = (hex: string, at: number) => parseInt(hex.slice(at, at + 2), 16)
    const out = [1, 3, 5].map((at) => Math.round(channel(a, at) + (channel(b, at) - channel(a, at)) * amount))

    return `#${out.map((value) => value.toString(16).padStart(2, "0")).join("")}`
}
