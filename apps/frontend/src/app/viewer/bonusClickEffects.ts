import * as THREE from "three"
import type {SpreadClick} from "../../backends/backend.ts"
import {tilePointSize} from "./pointSize.ts"

import markVertex from "./shaders/enclosureMark/vertex.glsl"
import markFragment from "./shaders/enclosureMark/fragment.glsl"
import waveVertex from "./shaders/enclosureWave/vertex.glsl"
import waveFragment from "./shaders/enclosureWave/fragment.glsl"

/**
 * What a click made under a bonus looks like, on every screen on the planet.
 *
 * - **A spread click** bursts green at the tile clicked and throws a spark onto
 *   each tile around it, one after the other round the circle, which pops as it
 *   lands. A ring runs out under them.
 * - **A boosted click** (triple clicks) is fast and electric: a cyan flash, three
 *   streaks shooting out, and three rings snapping out one after another — one
 *   per click the bonus is worth.
 *
 * It borrows the enclosure's shaders and its approach: the curves are plain
 * TypeScript written into attributes per frame, so the timing is tested rather
 * than tuned by eye in GLSL. A few sparks per click is nothing to move on the CPU.
 */

/** The mean arc between touching tiles, as the server measures it on this map. */
const TILE_SPACING = 0.004

export const SPREAD_THROW_FROM = 0.04
export const SPREAD_THROW_STAGGER = 0.035
export const SPREAD_TRAVEL_SECONDS = 0.2
export const SPREAD_LIFETIME_SECONDS = 1.2

export const BOOST_WAVES_AT = [0, 0.09, 0.18]
export const BOOST_STREAKS = 3
export const BOOST_TRAVEL_SECONDS = 0.3
export const BOOST_LIFETIME_SECONDS = 0.7

/** Everything fades over the last part of its life. */
const FADE_SHARE = 0.4

/** Above the tiles, which sit on the unit sphere, so the marks are never inside them. */
const LIFT = 1.003

/** A mark is never drawn smaller than this, however far out the camera is. */
const MIN_MARK_PX = 8

/** More than this at once and the oldest ends early: boosted players click fast. */
const MAX_PLAYING = 32

const GREEN = new THREE.Color(0.3, 1.0, 0.45)
const CYAN = new THREE.Color(0.35, 0.85, 1.0)

export type Spark = {
    /** Where it starts and where it ends, on the unit sphere. The same point for a spark that does not fly. */
    from: THREE.Vector3
    to: THREE.Vector3
    /** Seconds after the effect starts that it appears. */
    start: number
    /** Seconds it takes to fly from `from` to `to`. */
    travel: number
    /**
     * - `burst`: flashes in place, at the tile clicked.
     * - `landing`: flies onto a tile and pops there.
     * - `streak`: shoots out and is gone before it stops.
     */
    role: "burst" | "landing" | "streak"
}

export type Wave = {
    startsAt: number
    seconds: number
}

export type Choreography = {
    sparks: Spark[]
    waves: Wave[]
    /** The tile clicked, on the unit sphere. */
    centre: THREE.Vector3
    /** How far the rings run, in world units. */
    reach: number
    /** The rings are never smaller than this on screen. */
    minReachPx: number
    lifetime: number
    colour: THREE.Color
}

const at = (positions: ArrayLike<number>, tile: number) => new THREE.Vector3(
    positions[(tile - 1) * 3], positions[(tile - 1) * 3 + 1], positions[(tile - 1) * 3 + 2]).normalize()

/** Two directions along the ground at `centre`, to measure angles around it. */
function groundFrame(centre: THREE.Vector3): {east: THREE.Vector3, north: THREE.Vector3} {
    const east = new THREE.Vector3(0, 1, 0).cross(centre)
    if (east.lengthSq() < 1e-12) east.set(1, 0, 0)
    east.normalize()

    return {east, north: centre.clone().cross(east)}
}

/**
 * A spread click: a burst at the tile clicked, and one spark thrown onto each
 * tile around it, going round the circle.
 *
 * `positions` is the coordinates blob: three floats per tile, tile id - 1.
 */
export function choreographSpread(spread: SpreadClick, positions: ArrayLike<number>): Choreography {
    const centre = at(positions, spread.tile)
    const {east, north} = groundFrame(centre)

    // The server sends the neighbours in the map's order, which is no order on
    // the ground. Round the circle, the throws read as a spin.
    const around = spread.spread
        .map((tile) => {
            const p = at(positions, tile)
            return {p, angle: Math.atan2(p.dot(north), p.dot(east))}
        })
        .sort((a, b) => a.angle - b.angle)

    const sparks: Spark[] = [{from: centre, to: centre, start: 0, travel: 0, role: "burst"}]
    around.forEach(({p}, i) => sparks.push({
        from: centre,
        to: p,
        start: SPREAD_THROW_FROM + i * SPREAD_THROW_STAGGER,
        travel: SPREAD_TRAVEL_SECONDS,
        role: "landing",
    }))

    let radius = TILE_SPACING
    for (const {p} of around) radius = Math.max(radius, p.distanceTo(centre))

    return {
        sparks,
        waves: [{startsAt: 0.12, seconds: 0.8}, {startsAt: 0.3, seconds: 0.8}],
        centre,
        reach: radius * 5,
        minReachPx: 60,
        lifetime: SPREAD_LIFETIME_SECONDS,
        colour: GREEN,
    }
}

/**
 * A boosted click: a flash, three streaks out, three rings. The streaks are
 * turned by the tile id, so a run of clicks does not repeat the same star, and
 * the same click looks the same on every screen.
 */
export function choreographBoost(tile: number, positions: ArrayLike<number>): Choreography {
    const centre = at(positions, tile)
    const {east, north} = groundFrame(centre)

    const turn = (tile * 2.399963) % (Math.PI * 2)
    const sparks: Spark[] = [{from: centre, to: centre, start: 0, travel: 0, role: "burst"}]

    for (let i = 0; i < BOOST_STREAKS; i++) {
        const angle = turn + (i * Math.PI * 2) / BOOST_STREAKS
        const to = centre.clone()
            .addScaledVector(east, Math.cos(angle) * TILE_SPACING * 7)
            .addScaledVector(north, Math.sin(angle) * TILE_SPACING * 7)
            .normalize()
        sparks.push({from: centre, to, start: 0, travel: BOOST_TRAVEL_SECONDS, role: "streak"})
    }

    return {
        sparks,
        waves: BOOST_WAVES_AT.map((startsAt) => ({startsAt, seconds: 0.45})),
        centre,
        reach: TILE_SPACING * 8,
        minReachPx: 48,
        lifetime: BOOST_LIFETIME_SECONDS,
        colour: CYAN,
    }
}

export type SparkLook = {
    /** How far along from `from` to `to`, 0 to 1. */
    progress: number
    /** 0 is not drawn at all. */
    glow: number
    /** Of the mark's resting size. */
    scale: number
    /** How far the colour is washed to white: the flash. */
    white: number
}

const HIDDEN: SparkLook = {progress: 0, glow: 0, scale: 0, white: 0}

/**
 * How one spark looks `age` seconds into an effect that lasts `lifetime`.
 *
 * `calm` is for a player who asked for less motion: a landing spark is simply
 * there on its tile, streaks are not drawn, and nothing flashes.
 */
export function sparkLook(spark: Spark, age: number, lifetime: number, calm = false): SparkLook {
    const since = age - spark.start
    if (since < 0 || age >= lifetime) return HIDDEN

    const fade = 1 - smoothstep(lifetime * (1 - FADE_SHARE), lifetime, age)
    const flight = spark.travel > 0 ? Math.min(1, since / spark.travel) : 1
    const eased = 1 - Math.pow(1 - flight, 3)

    switch (spark.role) {
        case "burst": {
            const flash = calm ? 0 : Math.exp(-since * 8)
            return {progress: 0, glow: fade, scale: 1.6 + 3 * flash, white: 0.6 * flash}
        }
        case "landing": {
            if (calm) return {progress: 1, glow: 0.9 * fade, scale: 1.3, white: 0}
            if (flight < 1) return {progress: eased, glow: fade, scale: 1.4, white: 0.5}

            const pop = Math.exp(-(since - spark.travel) * 8)
            return {progress: 1, glow: 0.9 * fade, scale: 1.3 + 2 * pop, white: 0.4 * pop}
        }
        case "streak": {
            if (calm || flight >= 1) return HIDDEN
            return {progress: eased, glow: fade * (1 - smoothstep(0.6, 1, flight)), scale: 1.5 - 0.6 * flight, white: 0.7}
        }
    }
}

export type WaveLook = {
    /** Of the ring's full reach. */
    radius: number
    opacity: number
}

/** How a ring looks `age` seconds into the effect, or undefined while it is not running. */
export function waveLook(wave: Wave, age: number): WaveLook | undefined {
    const progress = (age - wave.startsAt) / wave.seconds
    if (progress < 0 || progress >= 1) return undefined

    const eased = 1 - Math.pow(1 - progress, 3)
    return {radius: 0.1 + 0.9 * eased, opacity: Math.pow(1 - progress, 2) * 0.85}
}

function smoothstep(from: number, to: number, x: number): number {
    const t = Math.min(1, Math.max(0, (x - from) / (to - from)))
    return t * t * (3 - 2 * t)
}

export type BonusClickEffects = {
    readonly object: THREE.Object3D
    /** Starts a spread click's effect on the next frame. */
    playSpread(spread: SpreadClick): void
    /** Starts the effect of a boosted click on `tile` on the next frame. */
    playBoost(tile: number): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number): void
    dispose(): void
}

type Playing = {
    choreography: Choreography
    /** Stamped on the first frame after it was played. */
    startedAt: number | undefined
    points: THREE.Points
    marks: THREE.ShaderMaterial
    position: THREE.BufferAttribute
    glow: THREE.BufferAttribute
    scale: THREE.BufferAttribute
    white: THREE.BufferAttribute
    waves: {mesh: THREE.Mesh, material: THREE.ShaderMaterial, wave: Wave}[]
}

export function createBonusClickEffects(positions: ArrayLike<number>): BonusClickEffects {
    const group = new THREE.Group()
    // After the tiles, which are transparent too: the marks sit over them.
    group.renderOrder = 1

    const plane = new THREE.CircleGeometry(1, 48)
    let playing: Playing[] = []
    const calm = prefersLessMotion()
    const between = new THREE.Vector3()

    const stop = (effect: Playing) => {
        group.remove(effect.points)
        effect.points.geometry.dispose()
        effect.marks.dispose()
        for (const wave of effect.waves) {
            group.remove(wave.mesh)
            wave.material.dispose()
        }
    }

    const play = (choreography: Choreography) => {
        // A hidden tab draws no frames, so everything it was sent would start at
        // once on its return. What happened while nobody watched is on the map.
        if (typeof document !== "undefined" && document.visibilityState === "hidden") return

        const count = choreography.sparks.length
        const geometry = new THREE.BufferGeometry()
        const position = new THREE.BufferAttribute(new Float32Array(count * 3), 3)
        const glow = new THREE.BufferAttribute(new Float32Array(count), 1)
        const scale = new THREE.BufferAttribute(new Float32Array(count), 1)
        const white = new THREE.BufferAttribute(new Float32Array(count), 1)
        geometry.setAttribute("position", position)
        geometry.setAttribute("glow", glow)
        geometry.setAttribute("scale", scale)
        geometry.setAttribute("white", white)

        const marks = new THREE.ShaderMaterial({
            uniforms: {
                tileSize: {value: 1},
                minSize: {value: MIN_MARK_PX},
                colour: {value: choreography.colour},
            },
            vertexShader: markVertex,
            fragmentShader: markFragment,
            transparent: true,
            depthWrite: false,
        })

        const points = new THREE.Points(geometry, marks)
        // The positions move every frame, so bounds computed once would be wrong.
        points.frustumCulled = false
        points.renderOrder = 1
        group.add(points)

        const waves = calm ? [] : choreography.waves.map((wave) => {
            const material = new THREE.ShaderMaterial({
                uniforms: {
                    colour: {value: choreography.colour},
                    radius: {value: 0},
                    opacity: {value: 0},
                },
                vertexShader: waveVertex,
                fragmentShader: waveFragment,
                transparent: true,
                depthWrite: false,
                // Not added: added light vanishes on the white of a flag, and
                // half the flags on the map have some.
                side: THREE.DoubleSide,
            })

            const mesh = new THREE.Mesh(plane, material)
            // Laid flat on the ground: a tangent plane never cuts into the sphere.
            mesh.position.copy(choreography.centre).multiplyScalar(LIFT)
            mesh.lookAt(choreography.centre.clone().multiplyScalar(2))
            mesh.visible = false
            mesh.renderOrder = 1
            group.add(mesh)

            return {mesh, material, wave}
        })

        playing.push({choreography, startedAt: undefined, points, marks, position, glow, scale, white, waves})

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
            const {choreography} = effect

            if (age >= choreography.lifetime) {
                stop(effect)
                return false
            }

            effect.marks.uniforms.tileSize.value = tileSize
            choreography.sparks.forEach((spark, i) => {
                const look = sparkLook(spark, age, choreography.lifetime, calm)
                // Along the chord and back out to the ground, so a spark never
                // dips under the tiles on its way.
                between.lerpVectors(spark.from, spark.to, look.progress).normalize().multiplyScalar(LIFT)
                effect.position.setXYZ(i, between.x, between.y, between.z)
                effect.glow.setX(i, look.glow)
                effect.scale.setX(i, look.scale)
                effect.white.setX(i, look.white)
            })
            effect.position.needsUpdate = true
            effect.glow.needsUpdate = true
            effect.scale.needsUpdate = true
            effect.white.needsUpdate = true

            const reach = Math.max(choreography.reach, choreography.minReachPx / pixelsPerUnit)
            for (const {mesh, material, wave} of effect.waves) {
                const look = waveLook(wave, age)
                mesh.visible = look !== undefined
                if (!look) continue

                mesh.scale.setScalar(reach)
                material.uniforms.radius.value = look.radius
                material.uniforms.opacity.value = look.opacity
            }

            return true
        })
    }

    return {
        object: group,
        playSpread: (spread) => play(choreographSpread(spread, positions)),
        playBoost: (tile) => play(choreographBoost(tile, positions)),
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
