import * as THREE from "three"
import type {VirusSpread} from "../../backends/backend.ts"
import {BoardScale, EARTH_1_SPACING, scaledLength} from "../../domain/boardScale.ts"

import slimeVertex from "./shaders/virusSlime/vertex.glsl"
import slimeFragment from "./shaders/virusSlime/fragment.glsl"

// A virus's goo, from the tile it holds onto the tile it takes, and a cured tile's goo bursting. The timelines are
// TypeScript, written into attributes per frame, so the tests pin them; the goo itself is the fragment shader's.

// When a tendril's goo reaches the tile it takes, and a use's goo erupts out of its tile: the tile turns then, so
// globe.ts holds a virus_spread this long.
export const SLIME_IMPACT = 0.4

// From the impact to the goo sunk into the tile, where the infected tile's icon takes over.
export const SPLAT_SECONDS = 0.8

// A tendril that met a shield is back in its tile this long after the contact.
export const RECOIL_SECONDS = 0.45

// A cured tile's goo swells this long, then bursts.
export const POP_AT = 0.14
export const BURST_SECONDS = 0.9

// The tendrils of one tick and the bursts of one cure set out within this, so they do not move as one.
const JITTER_SECONDS = 0.06
const BURST_JITTER_SECONDS = 0.15

// The reach a tendril finds its source within, in tile spacings: past it the goo erupts out of the tile it takes.
export const SOURCE_REACH = 4

// A blob's least radius on screen, in CSS pixels: from orbit a tile is under a pixel.
const MIN_BLOB_PX = 5

// The goo settles at this radius on the tile it took, in tile spacings: over the flag and its membrane.
export const SETTLED = 0.47

const MAX_SLIMES = 128

// A tile cured this soon after its goo landed bursts out of that goo.
const DOOM_WITHIN = 0.3

export type SlimeKind = "ooze" | "erupt" | "recoil" | "burst"

export type Slime = {
    kind: SlimeKind
    // Tiles; the same tile for a goo that stays on one.
    from: number
    to: number
    start: number
    // Turns each one's droplets, bubbles and wobble.
    seed: number
    // Its tile is cured as the goo lands (the virus ended on that tick): it bursts there instead of sinking in.
    doomed?: boolean
}

// Radii and lengths are in tile spacings.
export type SlimeLook = {
    // The head's place from the source (0) to the target (1).
    along: number
    head: number
    source: number
    // The goo between the source and the head: 0 once it snapped.
    neck: number
    drops: number
    drop: number
    wobble: number
    // 0: the droplets fly ahead of the head; 1: all round it.
    spray: number
    alpha: number
    // 0: live goo; 1: a dead grey husk.
    dead: number
    // 1: a glossy dome; 0: a flat stain.
    gloss: number
    flash: number
}

const LIVE: Omit<SlimeLook, "along" | "head" | "alpha"> = {
    source: 0, neck: 0, drops: 0, drop: 0, wobble: 0, spray: 0, dead: 0, gloss: 1, flash: 0,
}

export function slimeLifetime(kind: SlimeKind, calm = false, doomed = false): number {
    if (calm && kind === "recoil") return 0
    switch (kind) {
        case "ooze":
        case "erupt":
            if (!doomed) return SLIME_IMPACT + SPLAT_SECONDS
            return SLIME_IMPACT + (calm ? BURST_SECONDS : BURST_SECONDS - POP_AT)
        case "recoil":
            return SLIME_IMPACT + RECOIL_SECONDS + 0.2
        case "burst":
            return BURST_SECONDS
    }
}

// What a slime looks like `age` seconds into its effect; undefined before a tendril sets out and once it is over.
export function slimeLook(slime: Slime, age: number, calm = false): SlimeLook | undefined {
    const lifetime = slimeLifetime(slime.kind, calm, slime.doomed)
    if (age >= slime.start + lifetime) return undefined

    // A cured tile's goo is whole until it bursts: the tile's icon is already gone.
    const since = slime.kind === "burst" ? Math.max(0, age - slime.start) : age - slime.start
    if (since < 0) return undefined

    if (calm && slime.doomed) return since < SLIME_IMPACT ? undefined : calmLook("burst", since - SLIME_IMPACT)
    if (calm) return calmLook(slime.kind, since)

    switch (slime.kind) {
        case "ooze":
            if (since < SLIME_IMPACT) return reaching(since, 1)
            return slime.doomed ? landingDead(since - SLIME_IMPACT, true) : splatting(since - SLIME_IMPACT, true)
        case "erupt":
            if (since < SLIME_IMPACT) return erupting(since)
            return slime.doomed ? landingDead(since - SLIME_IMPACT, false) : splatting(since - SLIME_IMPACT, false)
        case "recoil":
            return since < SLIME_IMPACT ? reaching(since, 0.5) : recoiling(since - SLIME_IMPACT)
        case "burst":
            return bursting(since)
    }
}

// The goo gathers in its tile, then a tendril stretches out of it to `to` of the way.
function reaching(t: number, to: number): SlimeLook {
    const u = t / SLIME_IMPACT
    const along = to * easeInOut(clamp01((u - 0.15) / 0.85))
    return {
        ...LIVE,
        along,
        head: 0.26 + 0.05 * u,
        source: 0.3 + 0.1 * smoothstep(0, 0.25, u),
        neck: 0.15 * (1 - 0.35 * along),
        wobble: 0.6,
        alpha: 0.95 * smoothstep(0, 0.15, u),
    }
}

// A bubble of goo boils up out of the tile the virus was used on.
function erupting(t: number): SlimeLook {
    const u = t / SLIME_IMPACT
    return {...LIVE, along: 1, head: 0.05 + 0.27 * u * u, wobble: 1.4, alpha: 0.95 * smoothstep(0, 0.2, u)}
}

// The goo lands on its tile: it overshoots and jiggles, throws droplets, lets go of its source, and sinks in.
function splatting(s: number, fromSource: boolean): SlimeLook {
    const sink = smoothstep(0.35, SPLAT_SECONDS, s)
    return {
        along: 1,
        head: (SETTLED + 0.2 * Math.exp(-6 * s) * Math.cos(15 * s)) * (1 - 0.2 * sink),
        source: fromSource ? 0.4 * (1 - smoothstep(0.05, 0.4, s)) : 0,
        neck: fromSource ? 0.12 * (1 - smoothstep(0, 0.3, s)) : 0,
        drops: 0.35 + 0.55 * (1 - Math.exp(-7 * s)),
        drop: 0.11 * (1 - smoothstep(0.15, 0.55, s)),
        wobble: 0.3 + 1.6 * Math.exp(-5 * s),
        spray: fromSource ? 0 : 1,
        alpha: 0.95 * (1 - sink),
        dead: 0,
        gloss: 1,
        flash: 0.5 * Math.exp(-14 * s),
    }
}

// The goo lands on a tile cured as it lands: it bursts at once, and lets go of its source as a splat does.
function landingDead(s: number, fromSource: boolean): SlimeLook {
    const {source, neck} = splatting(s, fromSource)
    return {...bursting(POP_AT + s), source, neck}
}

// The tendril hit a shield halfway: it squashes, sprays a little, and pulls back into its tile.
function recoiling(s: number): SlimeLook {
    const back = easeOut(clamp01(s / RECOIL_SECONDS))
    const gone = smoothstep(RECOIL_SECONDS - 0.1, RECOIL_SECONDS + 0.2, s)
    return {
        along: 0.5 * (1 - back),
        head: 0.31 - 0.08 * back,
        source: 0.4,
        neck: 0.1 * (1 - back),
        drops: 0.2 + 0.3 * (1 - Math.exp(-8 * s)),
        drop: 0.07 * (1 - smoothstep(0.1, 0.35, s)),
        wobble: 0.6 + 1.8 * Math.exp(-7 * s),
        spray: 0,
        alpha: 0.95 * (1 - gone),
        dead: 0,
        gloss: 1,
        flash: 0,
    }
}

// The goo of a cured tile swells and trembles, bursts into droplets, and dries to a grey stain that fades.
function bursting(t: number): SlimeLook {
    if (t < POP_AT) return {...LIVE, along: 1, head: SETTLED + 0.12 * (t / POP_AT), wobble: 1.3, alpha: 0.92}
    const p = t - POP_AT
    return {
        along: 1,
        head: (SETTLED + 0.12) * (1 - 0.35 * smoothstep(0, 0.12, p)),
        source: 0,
        neck: 0,
        drops: 0.4 + 0.75 * (1 - Math.exp(-6 * p)),
        drop: 0.13 * (1 - smoothstep(0.2, 0.6, p)),
        wobble: 1.5 * Math.exp(-8 * p),
        spray: 1,
        alpha: 0.92 * (1 - smoothstep(0.2, BURST_SECONDS - POP_AT, p)),
        dead: smoothstep(0.05, 0.35, p),
        gloss: 1 - smoothstep(0, 0.2, p),
        flash: 0.7 * Math.exp(-12 * p),
    }
}

// With less motion nothing travels, jiggles or flies: the goo fades in on its tile, and a cured tile's goo dries and
// fades.
function calmLook(kind: SlimeKind, t: number): SlimeLook | undefined {
    switch (kind) {
        case "ooze":
        case "erupt":
            if (t < SLIME_IMPACT) return undefined
            return {...LIVE, along: 1, head: SETTLED, alpha: 0.9 * (1 - smoothstep(0, SPLAT_SECONDS, t - SLIME_IMPACT))}
        case "recoil":
            return undefined
        case "burst":
            return {
                ...LIVE,
                along: 1,
                head: SETTLED,
                alpha: 0.9 * (1 - smoothstep(0.2, BURST_SECONDS, t)),
                dead: smoothstep(0, 0.4, t),
                gloss: 1 - smoothstep(0, 0.4, t),
            }
    }
}

type Positions = ArrayLike<number>

function distanceBetween(positions: Positions, a: number, b: number): number {
    const i = (a - 1) * 3
    const j = (b - 1) * 3
    const ax = positions[i], ay = positions[i + 1], az = positions[i + 2]
    const bx = positions[j], by = positions[j + 1], bz = positions[j + 2]
    const la = Math.hypot(ax, ay, az) || 1
    const lb = Math.hypot(bx, by, bz) || 1
    return Math.hypot(ax / la - bx / lb, ay / la - by / lb, az / la - bz / lb)
}

// The tile of the virus nearest to `tile`, within `reach`: the one its goo comes from. The server picks a neighbour of
// one of its tiles; any of them nearest is as good a source.
export function sourceOf(tile: number, holding: readonly number[], positions: Positions, reach: number): number | undefined {
    let best: number | undefined
    let nearest = reach
    for (const from of holding) {
        if (from === tile) continue
        const distance = distanceBetween(positions, tile, from)
        if (distance <= nearest) {
            nearest = distance
            best = from
        }
    }
    return best
}

// A virus's use or tick as goo: a tendril from the nearest tile it holds onto each tile it took and each shield it struck,
// and goo erupting out of a tile it took with none of its tiles near (its use).
export function choreographSpread(
    {taken, struck}: Pick<VirusSpread, "taken" | "struck">,
    holding: readonly number[],
    positions: Positions,
    scale: BoardScale,
    random: () => number = Math.random,
): Slime[] {
    const reach = SOURCE_REACH * scaledLength(EARTH_1_SPACING, scale)
    const slimes: Slime[] = []
    for (const tile of taken) {
        const from = sourceOf(tile, holding, positions, reach)
        slimes.push({
            kind: from === undefined ? "erupt" : "ooze",
            from: from ?? tile,
            to: tile,
            start: from === undefined ? 0 : random() * JITTER_SECONDS,
            seed: random() * 100,
        })
    }
    for (const {tile} of struck) {
        const from = sourceOf(tile, holding, positions, reach)
        if (from === undefined) continue
        slimes.push({kind: "recoil", from, to: tile, start: random() * JITTER_SECONDS, seed: random() * 100})
    }
    return slimes
}

export function choreographCures(tiles: readonly number[], random: () => number = Math.random): Slime[] {
    const jitter = tiles.length > 1 ? BURST_JITTER_SECONDS : 0
    return tiles.map((tile) => ({kind: "burst", from: tile, to: tile, start: random() * jitter, seed: random() * 100}))
}

// The goo landing on a tile cured now bursts there, on landing: the tiles left to burst on their own. A goo that sank
// in long ago has no say.
export function doom(playing: (Slime & {startedAt: number | undefined})[], tiles: readonly number[], seconds: number): number[] {
    const left = new Set(tiles)
    for (const slime of playing) {
        if ((slime.kind !== "ooze" && slime.kind !== "erupt") || !left.has(slime.to)) continue
        const age = slime.startedAt === undefined ? 0 : seconds - slime.startedAt
        if (age - slime.start > SLIME_IMPACT + DOOM_WITHIN) continue
        slime.doomed = true
        left.delete(slime.to)
    }
    return tiles.filter((tile) => left.has(tile))
}

export type VirusSlime = {
    readonly object: THREE.Object3D
    // A virus_spread as it arrives: the tiles it takes turn SLIME_IMPACT later. `holding` is every tile the virus infects.
    spread(spread: Pick<VirusSpread, "taken" | "struck">, holding: readonly number[]): void
    // Tiles a virus infects no more.
    cure(tiles: readonly number[]): void
    prime(): void
    update(seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number): boolean
    dispose(): void
}

type Playing = Slime & {startedAt: number | undefined}

export function createVirusSlime(positions: Positions, scale: BoardScale, lift: THREE.IUniform<number>): VirusSlime {
    const spacing = scaledLength(EARTH_1_SPACING, scale)
    const calm = prefersLessMotion()

    const plane = new THREE.PlaneGeometry(1, 1)
    const geometry = new THREE.InstancedBufferGeometry()
    geometry.index = plane.index
    geometry.setAttribute("position", plane.getAttribute("position"))

    const attribute = (size: number) => {
        const values = new THREE.InstancedBufferAttribute(new Float32Array(MAX_SLIMES * size), size)
        values.setUsage(THREE.DynamicDrawUsage)
        return values
    }
    const from = attribute(3)
    const to = attribute(3)
    const shape = attribute(4)
    const splash = attribute(4)
    const tone = attribute(4)
    const seed = attribute(1)
    geometry.setAttribute("from", from)
    geometry.setAttribute("to", to)
    geometry.setAttribute("shape", shape)
    geometry.setAttribute("splash", splash)
    geometry.setAttribute("tone", tone)
    geometry.setAttribute("seed", seed)
    geometry.instanceCount = 0

    const uniforms = {
        time: {value: 0},
        spacing: {value: spacing},
        grow: {value: 1},
        unit: {value: 0.05},
        lift,
    }
    const material = new THREE.ShaderMaterial({
        uniforms,
        vertexShader: slimeVertex,
        fragmentShader: slimeFragment,
        transparent: true,
        depthWrite: false,
        side: THREE.DoubleSide,
    })

    const mesh = new THREE.Mesh(geometry, material)
    mesh.frustumCulled = false
    mesh.renderOrder = 1

    let playing: Playing[] = []
    let priming = false
    let drawn = false
    let clock = 0

    const play = (slimes: Slime[]) => {
        if (slimes.length === 0) return
        if (typeof document !== "undefined" && document.visibilityState === "hidden") return
        for (const slime of slimes) {
            if (slimeLifetime(slime.kind, calm, slime.doomed) > 0) playing.push({...slime, startedAt: undefined})
        }
        if (playing.length > MAX_SLIMES) playing = playing.slice(playing.length - MAX_SLIMES)
    }

    const setTile = (values: THREE.InstancedBufferAttribute, index: number, tile: number) => {
        const i = (tile - 1) * 3
        values.setXYZ(index, positions[i], positions[i + 1], positions[i + 2])
    }

    const update = (seconds: number, camera: THREE.OrthographicCamera, viewportHeight: number, pixelRatio: number) => {
        if (priming) {
            priming = false
            setTile(from, 0, 1)
            setTile(to, 0, 1)
            tone.setXYZW(0, 0, 0, 0, 0)
            from.needsUpdate = to.needsUpdate = tone.needsUpdate = true
            geometry.instanceCount = 1
            drawn = true
            return true
        }
        if (playing.length === 0) {
            if (!drawn) return false
            drawn = false
            geometry.instanceCount = 0
            return true
        }

        clock = seconds
        const spacingPx = spacing * (viewportHeight / 2) * camera.zoom
        uniforms.time.value = seconds
        uniforms.grow.value = Math.max(1, (MIN_BLOB_PX * pixelRatio) / (SETTLED * spacingPx))
        uniforms.unit.value = 1 / Math.max(spacingPx, 1e-6)

        let count = 0
        playing = playing.filter((slime) => {
            slime.startedAt ??= seconds
            const age = seconds - slime.startedAt
            if (age >= slime.start + slimeLifetime(slime.kind, calm, slime.doomed)) return false

            const look = slimeLook(slime, age, calm)
            setTile(from, count, slime.from)
            setTile(to, count, slime.to)
            seed.setX(count, slime.seed)
            if (look) {
                shape.setXYZW(count, look.along, look.head, look.source, look.neck)
                splash.setXYZW(count, look.drops, look.drop, look.wobble, look.spray)
                tone.setXYZW(count, look.alpha, look.dead, look.gloss, look.flash)
            } else {
                tone.setXYZW(count, 0, 0, 0, 0)
            }
            count++
            return true
        })

        for (const values of [from, to, shape, splash, tone, seed]) {
            values.clearUpdateRanges()
            values.addUpdateRange(0, count * values.itemSize)
            values.needsUpdate = true
        }
        geometry.instanceCount = count
        drawn = count > 0
        return true
    }

    return {
        object: mesh,
        spread: (spread, holding) => play(choreographSpread(spread, holding, positions, scale)),
        cure: (tiles) => play(choreographCures(doom(playing, tiles, clock))),
        prime: () => {
            priming = true
        },
        update,
        dispose: () => {
            playing = []
            geometry.dispose()
            plane.dispose()
            material.dispose()
        },
    }
}

function clamp01(x: number): number {
    return Math.min(1, Math.max(0, x))
}

function smoothstep(from: number, to: number, x: number): number {
    const t = clamp01((x - from) / (to - from))
    return t * t * (3 - 2 * t)
}

function easeInOut(t: number): number {
    return t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
}

function easeOut(t: number): number {
    return 1 - Math.pow(1 - t, 3)
}

function prefersLessMotion(): boolean {
    return typeof window !== "undefined"
        && typeof window.matchMedia === "function"
        && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}
