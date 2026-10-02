import * as THREE from "three"
import type {Enclosure} from "../../backends/backend.ts"
import {tilePointSize} from "./pointSize.ts"

import markVertex from "./shaders/enclosureMark/vertex.glsl"
import markFragment from "./shaders/enclosureMark/fragment.glsl"
import waveVertex from "./shaders/enclosureWave/vertex.glsl"
import waveFragment from "./shaders/enclosureWave/fragment.glsl"

export const OUTLINE_SECONDS = 0.5

export const FILL_FROM = 0.5
export const FILL_SECONDS = 0.45

export const WAVES_AT = [0.55, 0.8]
export const WAVE_SECONDS = 1.1

export const FADE_FROM = 1.9
export const LIFETIME_SECONDS = 2.6

const LIFT = 1.003

const MIN_MARK_PX = 5

const MIN_WAVE_PX = 42

const WAVE_REACH = 3.5

const MAX_PLAYING = 12

const GOLD = new THREE.Color(1.0, 0.68, 0.16)

export type Mark = {
    tile: number
    start: number
    role: "wall" | "closing" | "filled"
}

export type Choreography = {
    marks: Mark[]
    centre: THREE.Vector3
    radius: number
}

export function choreograph(enclosure: Enclosure, positions: ArrayLike<number>): Choreography {
    const at = (tile: number) => new THREE.Vector3(
        positions[(tile - 1) * 3], positions[(tile - 1) * 3 + 1], positions[(tile - 1) * 3 + 2])

    const tiles = [...enclosure.wall, ...enclosure.filled]
    const centre = new THREE.Vector3()
    for (const tile of tiles) centre.add(at(tile))
    if (centre.lengthSq() === 0) centre.copy(at(enclosure.closingTile))
    centre.normalize()

    let radius = 0
    for (const tile of tiles) radius = Math.max(radius, at(tile).distanceTo(centre))

    const east = new THREE.Vector3(0, 1, 0).cross(centre)
    if (east.lengthSq() < 1e-12) east.set(1, 0, 0)
    east.normalize()
    const north = centre.clone().cross(east)

    const angleOf = (tile: number) => {
        const p = at(tile)
        return Math.atan2(p.dot(north), p.dot(east))
    }

    const closingAngle = angleOf(enclosure.closingTile)
    const marks: Mark[] = []

    for (const tile of enclosure.wall) {
        if (tile === enclosure.closingTile) {
            marks.push({tile, start: OUTLINE_SECONDS, role: "closing"})
            continue
        }

        const turn = Math.abs(normaliseAngle(angleOf(tile) - closingAngle)) / Math.PI
        marks.push({tile, start: OUTLINE_SECONDS * (1 - turn), role: "wall"})
    }

    enclosure.filled.forEach((tile, index) => {
        const share = enclosure.filled.length > 1 ? index / (enclosure.filled.length - 1) : 0
        marks.push({tile, start: FILL_FROM + FILL_SECONDS * share, role: "filled"})
    })

    return {marks, centre, radius}
}

function normaliseAngle(angle: number): number {
    return Math.atan2(Math.sin(angle), Math.cos(angle))
}

export type MarkLook = {
    glow: number
    scale: number
    white: number
}

const HIDDEN: MarkLook = {glow: 0, scale: 0, white: 0}

export function markLook(mark: Mark, age: number, calm = false): MarkLook {
    const since = age - mark.start
    if (since < 0 || age >= LIFETIME_SECONDS) return HIDDEN

    const fade = 1 - smoothstep(FADE_FROM, LIFETIME_SECONDS, age)
    const flash = calm ? 0 : Math.exp(-since * 6)

    switch (mark.role) {
        case "wall":
            return {glow: Math.min(1, 0.85 + flash) * fade, scale: 1.15 + 0.8 * flash, white: flash}
        case "closing":
            return {glow: fade, scale: 1.2 + 2.4 * flash, white: flash}
        case "filled": {
            const pop = calm ? 0 : Math.exp(-since * 7)
            return {glow: 0.8 * fade, scale: 1.05 + 1.6 * pop, white: 0.4 + 0.6 * flash}
        }
    }
}

export type WaveLook = {
    radius: number
    opacity: number
}

export function waveLook(startsAt: number, age: number): WaveLook | undefined {
    const progress = (age - startsAt) / WAVE_SECONDS
    if (progress < 0 || progress >= 1) return undefined

    const eased = 1 - Math.pow(1 - progress, 3)
    return {radius: 0.12 + 0.88 * eased, opacity: Math.pow(1 - progress, 2) * 0.9}
}

function smoothstep(from: number, to: number, x: number): number {
    const t = Math.min(1, Math.max(0, (x - from) / (to - from)))
    return t * t * (3 - 2 * t)
}

export type EnclosureEffects = {
    readonly object: THREE.Object3D
    play(enclosure: Enclosure): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number): boolean
    dispose(): void
}

type Playing = {
    choreography: Choreography
    startedAt: number | undefined
    points: THREE.Points
    marks: THREE.ShaderMaterial
    glow: THREE.BufferAttribute
    scale: THREE.BufferAttribute
    white: THREE.BufferAttribute
    waves: {mesh: THREE.Mesh, material: THREE.ShaderMaterial, startsAt: number}[]
}

const waveGeometry = () => new THREE.CircleGeometry(1, 64)

export function createEnclosureEffects(positions: ArrayLike<number>): EnclosureEffects {
    const group = new THREE.Group()
    group.renderOrder = 1

    const plane = waveGeometry()
    let playing: Playing[] = []
    const calm = prefersLessMotion()

    const stop = (effect: Playing) => {
        group.remove(effect.points)
        effect.points.geometry.dispose()
        effect.marks.dispose()
        for (const wave of effect.waves) {
            group.remove(wave.mesh)
            wave.material.dispose()
        }
    }

    const play = (enclosure: Enclosure) => {
        if (typeof document !== "undefined" && document.visibilityState === "hidden") return

        const choreography = choreograph(enclosure, positions)
        const {marks} = choreography

        const geometry = new THREE.BufferGeometry()
        const lifted = new Float32Array(marks.length * 3)
        marks.forEach(({tile}, i) => {
            for (let axis = 0; axis < 3; axis++) lifted[i * 3 + axis] = positions[(tile - 1) * 3 + axis] * LIFT
        })
        geometry.setAttribute("position", new THREE.BufferAttribute(lifted, 3))

        const glow = new THREE.BufferAttribute(new Float32Array(marks.length), 1)
        const scale = new THREE.BufferAttribute(new Float32Array(marks.length), 1)
        const white = new THREE.BufferAttribute(new Float32Array(marks.length), 1)
        geometry.setAttribute("glow", glow)
        geometry.setAttribute("scale", scale)
        geometry.setAttribute("white", white)

        const material = new THREE.ShaderMaterial({
            uniforms: {
                tileSize: {value: 1},
                minSize: {value: MIN_MARK_PX},
                unitsPerPixel: {value: 0},
                colour: {value: GOLD},
            },
            vertexShader: markVertex,
            fragmentShader: markFragment,
            transparent: true,
            depthWrite: false,
        })

        const points = new THREE.Points(geometry, material)
        points.frustumCulled = false
        points.renderOrder = 1
        group.add(points)

        const waves = calm ? [] : WAVES_AT.map((startsAt) => {
            const waveMaterial = new THREE.ShaderMaterial({
                uniforms: {
                    colour: {value: GOLD},
                    radius: {value: 0},
                    opacity: {value: 0},
                },
                vertexShader: waveVertex,
                fragmentShader: waveFragment,
                transparent: true,
                depthWrite: false,
                blending: THREE.AdditiveBlending,
                side: THREE.DoubleSide,
            })

            const mesh = new THREE.Mesh(plane, waveMaterial)
            mesh.position.copy(choreography.centre).multiplyScalar(LIFT)
            mesh.lookAt(choreography.centre.clone().multiplyScalar(2))
            mesh.visible = false
            mesh.renderOrder = 1
            group.add(mesh)

            return {mesh, material: waveMaterial, startsAt}
        })

        playing.push({choreography, startedAt: undefined, points, marks: material, glow, scale, white, waves})

        while (playing.length > MAX_PLAYING) {
            const oldest = playing.shift()
            if (oldest) stop(oldest)
        }
    }

    const update = (seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number) => {
        if (playing.length === 0) return false

        const tileSize = tilePointSize(camera.zoom, viewportHeight)
        const minMark = MIN_MARK_PX * pixelRatio
        const pixelsPerUnit = (viewportHeight / 2) * camera.zoom

        playing = playing.filter((effect) => {
            effect.startedAt ??= seconds
            const age = seconds - effect.startedAt

            if (age >= LIFETIME_SECONDS) {
                stop(effect)
                return false
            }

            effect.marks.uniforms.tileSize.value = tileSize
            effect.marks.uniforms.minSize.value = minMark
            effect.marks.uniforms.unitsPerPixel.value = 1 / pixelsPerUnit
            effect.choreography.marks.forEach((mark, i) => {
                const look = markLook(mark, age, calm)
                effect.glow.setX(i, look.glow)
                effect.scale.setX(i, look.scale)
                effect.white.setX(i, look.white)
            })
            effect.glow.needsUpdate = true
            effect.scale.needsUpdate = true
            effect.white.needsUpdate = true

            const reach = Math.max(effect.choreography.radius * WAVE_REACH, MIN_WAVE_PX * pixelRatio / pixelsPerUnit)
            for (const wave of effect.waves) {
                const look = waveLook(wave.startsAt, age)
                wave.mesh.visible = look !== undefined
                if (!look) continue

                wave.mesh.scale.setScalar(reach)
                wave.material.uniforms.radius.value = look.radius
                wave.material.uniforms.opacity.value = look.opacity
            }

            return true
        })

        return true
    }

    return {
        object: group,
        play,
        update,
        dispose: () => {
            for (const effect of playing) stop(effect)
            playing = []
            plane.dispose()
        },
    }
}

function prefersLessMotion(): boolean {
    return typeof window !== "undefined"
        && typeof window.matchMedia === "function"
        && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}
