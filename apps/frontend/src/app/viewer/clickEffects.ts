import * as THREE from "three"
import type {SpreadClick} from "../../backends/backend.ts"
import {tilePointSize} from "./pointSize.ts"

import markVertex from "./shaders/enclosureMark/vertex.glsl"
import markFragment from "./shaders/enclosureMark/fragment.glsl"
import waveVertex from "./shaders/enclosureWave/vertex.glsl"
import waveFragment from "./shaders/enclosureWave/fragment.glsl"

const TILE_SPACING = 0.004

export const SPREAD_THROW_FROM = 0.04
export const SPREAD_THROW_STAGGER = 0.035
export const SPREAD_TRAVEL_SECONDS = 0.2
export const SPREAD_LIFETIME_SECONDS = 1.2

const FADE_SHARE = 0.4

const LIFT = 1.003

const MIN_MARK_PX = 8

const MAX_PLAYING = 32

const GREEN = new THREE.Color(0.3, 1.0, 0.45)

export const CLEAR_DRIFT_SECONDS = 0.45
export const CLEAR_LIFETIME_SECONDS = 0.8

const CLEAR_DRIFT_REACH = 0.9

const DUST = new THREE.Color(0.93, 0.8, 0.58)

export const CLICK_LIFETIME_SECONDS = 0.7

const CLICK_PEAK = 1

const CLICK_RIM = 0.6

const SKY = new THREE.Color(0.35, 0.75, 1.0)

export type Spark = {
    from: THREE.Vector3
    to: THREE.Vector3
    start: number
    travel: number
    role: "burst" | "landing" | "mote"
}

export type Wave = {
    startsAt: number
    seconds: number
    peak: number
}

export type Choreography = {
    sparks: Spark[]
    waves: Wave[]
    centre: THREE.Vector3
    reach: number
    minReachPx: number
    lifetime: number
    colour: THREE.Color
    rim: number
}

const at = (positions: ArrayLike<number>, tile: number) => new THREE.Vector3(
    positions[(tile - 1) * 3], positions[(tile - 1) * 3 + 1], positions[(tile - 1) * 3 + 2]).normalize()

function groundFrame(centre: THREE.Vector3): {east: THREE.Vector3, north: THREE.Vector3} {
    const east = new THREE.Vector3(0, 1, 0).cross(centre)
    if (east.lengthSq() < 1e-12) east.set(1, 0, 0)
    east.normalize()

    return {east, north: centre.clone().cross(east)}
}

export function choreographSpread(spread: SpreadClick, positions: ArrayLike<number>): Choreography {
    const centre = at(positions, spread.tile)
    const {east, north} = groundFrame(centre)

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
        waves: [{startsAt: 0.12, seconds: 0.8, peak: 0.85}, {startsAt: 0.3, seconds: 0.8, peak: 0.85}],
        centre,
        reach: radius * 5,
        minReachPx: 60,
        lifetime: SPREAD_LIFETIME_SECONDS,
        colour: GREEN,
        rim: 0,
    }
}

export function choreographClear(tile: number, positions: ArrayLike<number>): Choreography {
    const centre = at(positions, tile)
    const {east, north} = groundFrame(centre)

    const sparks: Spark[] = [{from: centre, to: centre, start: 0, travel: 0, role: "burst"}]
    for (let i = 0; i < 6; i++) {
        const angle = (i + 0.5) * Math.PI / 3
        const to = centre.clone()
            .addScaledVector(east, Math.cos(angle) * TILE_SPACING * CLEAR_DRIFT_REACH)
            .addScaledVector(north, Math.sin(angle) * TILE_SPACING * CLEAR_DRIFT_REACH)
            .normalize()
        sparks.push({from: centre, to, start: 0.02, travel: CLEAR_DRIFT_SECONDS, role: "mote"})
    }

    return {
        sparks,
        waves: [{startsAt: 0.04, seconds: 0.55, peak: 0.85}],
        centre,
        reach: TILE_SPACING * 3,
        minReachPx: 36,
        lifetime: CLEAR_LIFETIME_SECONDS,
        colour: DUST,
        rim: 0,
    }
}

// Every click plays this, so it stays plainer than any bonus: one ring, no spark. Plain, not faint.
export function choreographClick(tile: number, positions: ArrayLike<number>): Choreography {
    return {
        sparks: [],
        waves: [{startsAt: 0, seconds: CLICK_LIFETIME_SECONDS, peak: CLICK_PEAK}],
        centre: at(positions, tile),
        reach: TILE_SPACING * 2,
        minReachPx: 44,
        lifetime: CLICK_LIFETIME_SECONDS,
        colour: SKY,
        rim: CLICK_RIM,
    }
}

export function inView(point: THREE.Vector3, camera: THREE.Camera): boolean {
    if (point.dot(camera.getWorldDirection(new THREE.Vector3())) >= 0) return false

    const {x, y} = point.clone().project(camera)
    return Math.abs(x) <= 1 && Math.abs(y) <= 1
}

export type SparkLook = {
    progress: number
    glow: number
    scale: number
    white: number
}

const HIDDEN: SparkLook = {progress: 0, glow: 0, scale: 0, white: 0}

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
        case "mote": {
            if (calm) return {progress: 0.5, glow: 0.6 * fade, scale: 1, white: 0}
            const drift = 1 - Math.pow(1 - flight, 2)
            return {progress: drift, glow: fade * (1 - 0.7 * flight), scale: 1.2 - 0.6 * flight, white: 0.2 * (1 - flight)}
        }
    }
}

export type WaveLook = {
    radius: number
    opacity: number
}

export function waveLook(wave: Wave, age: number): WaveLook | undefined {
    const progress = (age - wave.startsAt) / wave.seconds
    if (progress < 0 || progress >= 1) return undefined

    const eased = 1 - Math.pow(1 - progress, 3)
    return {radius: 0.1 + 0.9 * eased, opacity: Math.pow(1 - progress, 2) * wave.peak}
}

function smoothstep(from: number, to: number, x: number): number {
    const t = Math.min(1, Math.max(0, (x - from) / (to - from)))
    return t * t * (3 - 2 * t)
}

export type ClickEffects = {
    readonly object: THREE.Object3D
    playSpread(spread: SpreadClick): void
    playClear(tile: number): void
    playClick(tile: number, camera: THREE.Camera): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number): boolean
    dispose(): void
}

type Playing = {
    choreography: Choreography
    startedAt: number | undefined
    points: THREE.Points
    marks: THREE.ShaderMaterial
    position: THREE.BufferAttribute
    glow: THREE.BufferAttribute
    scale: THREE.BufferAttribute
    white: THREE.BufferAttribute
    waves: {mesh: THREE.Mesh, material: THREE.ShaderMaterial, wave: Wave}[]
}

export function createClickEffects(positions: ArrayLike<number>): ClickEffects {
    const group = new THREE.Group()
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
        if (typeof document !== "undefined" && document.visibilityState === "hidden") return

        const rings = calm ? [] : choreography.waves
        if (choreography.sparks.length === 0 && rings.length === 0) return

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
        points.frustumCulled = false
        points.renderOrder = 1
        group.add(points)

        const waves = rings.map((wave) => {
            const material = new THREE.ShaderMaterial({
                uniforms: {
                    colour: {value: choreography.colour},
                    radius: {value: 0},
                    opacity: {value: 0},
                    rim: {value: choreography.rim},
                },
                vertexShader: waveVertex,
                fragmentShader: waveFragment,
                transparent: true,
                depthWrite: false,
                // Not additive: additive rings vanish on the white of a flag.
                side: THREE.DoubleSide,
            })

            const mesh = new THREE.Mesh(plane, material)
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

    const update = (seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number) => {
        if (playing.length === 0) return false

        const tileSize = tilePointSize(camera.zoom, viewportHeight)
        const minMark = MIN_MARK_PX * pixelRatio
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
            effect.marks.uniforms.minSize.value = minMark
            choreography.sparks.forEach((spark, i) => {
                const look = sparkLook(spark, age, choreography.lifetime, calm)
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

            const reach = Math.max(choreography.reach, choreography.minReachPx * pixelRatio / pixelsPerUnit)
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

        return true
    }

    return {
        object: group,
        playSpread: (spread) => play(choreographSpread(spread, positions)),
        playClear: (tile) => play(choreographClear(tile, positions)),
        playClick: (tile, camera) => {
            const choreography = choreographClick(tile, positions)
            if (inView(choreography.centre, camera)) play(choreography)
        },
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
