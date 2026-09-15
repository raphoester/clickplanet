import * as THREE from "three"
import type {Enclosure} from "../../backends/backend.ts"
import {tilePointSize} from "./pointSize.ts"

import markVertex from "./shaders/enclosureMark/vertex.glsl"
import markFragment from "./shaders/enclosureMark/fragment.glsl"
import waveVertex from "./shaders/enclosureWave/vertex.glsl"
import waveFragment from "./shaders/enclosureWave/fragment.glsl"

/**
 * What a closed shape looks like, on every screen on the planet.
 *
 * Tiles flipping in a patch all at once says nothing about why, so a shape tells
 * its story in three beats:
 *
 * 1. **The outline lights up**, running round both sides from the far end and
 *    meeting at the tile that closed it, which flashes. That is the shape
 *    snapping shut.
 * 2. **The inside pours in** from the closing tile, one tile after another.
 * 3. **A ring runs out over the ground** from the middle, so a shape closed far
 *    off, or seen from orbit where it is a few pixels across, is still noticed.
 *
 * Everything that moves is computed here, on the CPU, from plain functions of
 * time: a shape is a few dozen points, and keeping the curves in TypeScript is
 * what lets them be tested rather than tuned by eye in GLSL.
 */

/** The outline takes this long to run round and meet at the closing tile. */
export const OUTLINE_SECONDS = 0.5

/** The inside starts pouring in as the outline closes, and takes this long. */
export const FILL_FROM = 0.5
export const FILL_SECONDS = 0.45

/** Two rings, one after the other, each this long. */
export const WAVES_AT = [0.55, 0.8]
export const WAVE_SECONDS = 1.1

/** Everything holds a glow, then fades out over the tail of the effect. */
export const FADE_FROM = 1.9
export const LIFETIME_SECONDS = 2.6

/** Above the tiles, which sit on the unit sphere, so the marks are never inside them. */
const LIFT = 1.003

/** A mark is never drawn smaller than this, however far out the camera is. */
const MIN_MARK_PX = 5

/** Nor is a ring, which is what finds the shape for a player zoomed right out. */
const MIN_WAVE_PX = 42

/** How far the ring runs, in multiples of the shape's own radius. */
const WAVE_REACH = 3.5

/** More than this at once and the oldest ends early: a burst is still a burst. */
const MAX_PLAYING = 12

/** The bonus box's own gold, so a shape reads as something a box did. */
const GOLD = new THREE.Color(1.0, 0.68, 0.16)

export type Mark = {
    tile: number
    /** Seconds after the effect starts that this tile lights up. */
    start: number
    role: "wall" | "closing" | "filled"
}

export type Choreography = {
    marks: Mark[]
    /** The middle of the shape, on the unit sphere. */
    centre: THREE.Vector3
    /** How far the furthest tile of the shape is from its middle, in world units. */
    radius: number
}

/**
 * When each tile of a shape lights up, and where the shape is.
 *
 * `positions` is the coordinates blob: three floats per tile, tile id - 1.
 */
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

    // A frame on the ground at the middle of the shape, to measure the angle of
    // each outline tile around it.
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

        // How far round the outline from the closing tile, as a share of the
        // way to the far side: 0 beside it, 1 opposite. The far side lights
        // first, and the light runs both ways towards the closing tile.
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
    /** 0 is not drawn at all. */
    glow: number
    /** Of the mark's resting size. */
    scale: number
    /** How far the gold is washed to white: the flash. */
    white: number
}

const HIDDEN: MarkLook = {glow: 0, scale: 0, white: 0}

/**
 * How one mark looks `age` seconds into the effect.
 *
 * `calm` is for a player who asked for less motion: the shape still lights up
 * and fades, so what happened is still said, but nothing flashes or swells.
 */
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
            // Arrives big and white, and settles paler than the outline, so the
            // inside and the wall still read as two things.
            const pop = calm ? 0 : Math.exp(-since * 7)
            return {glow: 0.8 * fade, scale: 1.05 + 1.6 * pop, white: 0.4 + 0.6 * flash}
        }
    }
}

export type WaveLook = {
    /** Of the ring's full reach. */
    radius: number
    opacity: number
}

/** How a ring looks `age` seconds into the effect, or undefined while it is not running. */
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
    /** Starts a shape's effect on the next frame. */
    play(enclosure: Enclosure): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number): void
    dispose(): void
}

type Playing = {
    choreography: Choreography
    /** Stamped on the first frame after it was played: see play. */
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
    // After the tiles, which are transparent too: the marks sit over them.
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
        // A hidden tab draws no frames, so everything it was sent would start at
        // once on its return. What happened while nobody watched is on the map.
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
                colour: {value: GOLD},
            },
            vertexShader: markVertex,
            fragmentShader: markFragment,
            transparent: true,
            depthWrite: false,
        })

        const points = new THREE.Points(geometry, material)
        // Its bounds would be computed from the positions once and never again;
        // a shape is small and on screen or not as a whole, so skip the check.
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
            // Laid flat on the ground: a tangent plane never cuts into the sphere.
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

    const update = (seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number) => {
        if (playing.length === 0) return

        const tileSize = tilePointSize(camera.zoom, viewportHeight)
        // The camera's frustum is two units tall at zoom 1, so this is how many
        // pixels one world unit spans on screen right now.
        const pixelsPerUnit = (viewportHeight / 2) * camera.zoom

        playing = playing.filter((effect) => {
            effect.startedAt ??= seconds
            const age = seconds - effect.startedAt

            if (age >= LIFETIME_SECONDS) {
                stop(effect)
                return false
            }

            effect.marks.uniforms.tileSize.value = tileSize
            effect.choreography.marks.forEach((mark, i) => {
                const look = markLook(mark, age, calm)
                effect.glow.setX(i, look.glow)
                effect.scale.setX(i, look.scale)
                effect.white.setX(i, look.white)
            })
            effect.glow.needsUpdate = true
            effect.scale.needsUpdate = true
            effect.white.needsUpdate = true

            const reach = Math.max(effect.choreography.radius * WAVE_REACH, MIN_WAVE_PX / pixelsPerUnit)
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
