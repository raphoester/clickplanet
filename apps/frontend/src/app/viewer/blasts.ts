import * as THREE from "three"
import {BLAST_TIMELINE, blastOver, MAX_BLASTS} from "../../domain/blast.ts"
import {isBehindGlobe} from "./bonusBox.ts"

import debrisVertex from "./shaders/debris/vertex.glsl"
import debrisFragment from "./shaders/debris/fragment.glsl"
import ringVertex from "./shaders/ring/vertex.glsl"
import ringFragment from "./shaders/ring/fragment.glsl"

const DEBRIS_PER_BLAST = 140

/** How far past its edge a ring's quad reaches, so the soft edge is not cut. */
const RING_MARGIN = 1.2

/** How wide the incoming ring starts, in blast radii, before closing in. */
const INCOMING_REACH = 2.4

const Z_AXIS = new THREE.Vector3(0, 0, 1)

const TARGET_RED = new THREE.Color(1, 0.18, 0.1)
const SPLASH_RING = new THREE.Color(0.75, 0.93, 1)
const BLAST_FLASH = 0xffc070
const SPLASH_FLASH = 0xbfe8ff

/** A splash is smaller than a blast: a fountain, not a fireball. */
const SPLASH_SCALE = 0.6

const RIPPLE_SECONDS = 1.6

/** How long the flash at the centre lasts after impact, in seconds. */
const FLASH_SECONDS = 0.7

/** A slot nobody has used: dropped long enough ago that every phase is over. */
const NEVER = -1e6

/**
 * The uniforms the display shader reads the blasts from. They are merged into
 * the tile field's own uniforms, so the ground shakes in the same draw call
 * that paints it — no second pass over a million points.
 */
export type BlastUniforms = {
    time: THREE.IUniform<number>
    blasts: THREE.IUniform<THREE.Vector4[]>
    blastRadii: THREE.IUniform<number[]>
    motion: THREE.IUniform<number>
}

export function blastUniforms(reducedMotion: boolean): BlastUniforms {
    return {
        time: {value: 0},
        blasts: {value: Array.from({length: MAX_BLASTS}, () => new THREE.Vector4(0, 0, 1, NEVER))},
        blastRadii: {value: new Array(MAX_BLASTS).fill(0)},
        motion: {value: reducedMotion ? 0 : 1},
    }
}

export type Blasts = {
    readonly object: THREE.Object3D

    /**
     * Starts a blast at `centre` (any length; it is put on the unit sphere). A
     * splash is a bomb that fell in the sea: no crater, a ripple instead.
     */
    start(centre: THREE.Vector3, radius: number, seconds: number, splash?: boolean): void

    /** The aiming ring, or none. */
    setAim(centre: THREE.Vector3 | undefined, radius: number): void

    /** How far a held press has got towards dropping the bomb, 0 to 1. */
    setCharge(progress: number): void

    /** Where the newest blast still worth pointing at is, if any. */
    newest(seconds: number): THREE.Vector3 | undefined

    update(seconds: number, camera: THREE.Camera): void

    dispose(): void
}

type Slot = {
    centre: THREE.Vector3
    startedAt: number
    radius: number
    splash: boolean
    flash: THREE.Sprite
    debris: THREE.Points
    incoming: THREE.Mesh
}

/**
 * Everything a bomb draws, in a fixed number of slots allocated once.
 *
 * The ground effects — shock wave, scorch — live in the display shader and
 * cost a uniform write per frame. On top of them each slot owns a flash sprite,
 * a small debris cloud animated on the GPU from a start time, and the ring that
 * closes in while the bomb falls; a new blast reuses the oldest slot rather
 * than allocating. The aiming ring is one more ring of the same kind.
 *
 * The rings are meshes laid on the globe rather than drawn by the tiles: the
 * tiles are dots with sea and gaps between them, and a ring made of them breaks
 * up and flickers as it moves.
 */
export function createBlasts(uniforms: BlastUniforms, pixelsPerRadian: THREE.IUniform<number>): Blasts {
    const group = new THREE.Group()
    const flashTexture = glowTexture()
    const seeds = debrisSeeds()
    const ringGeometry = new THREE.PlaneGeometry(2, 2)
    const toCamera = new THREE.Vector3()
    const reducedMotion = uniforms.motion.value === 0

    const aim = createRing(ringGeometry)
    group.add(aim)

    const slots: Slot[] = Array.from({length: MAX_BLASTS}, () => {
        const flash = new THREE.Sprite(new THREE.SpriteMaterial({
            map: flashTexture,
            color: 0xffc070,
            blending: THREE.AdditiveBlending,
            depthWrite: false,
            transparent: true,
        }))
        flash.visible = false

        const geometry = new THREE.BufferGeometry()
        // Positions are computed in the shader; three still wants the attribute
        // to know how many points to draw.
        geometry.setAttribute("position", new THREE.BufferAttribute(new Float32Array(DEBRIS_PER_BLAST * 3), 3))
        geometry.setAttribute("seed", seeds)
        // Never culled on a stale bounding sphere of zeros.
        geometry.boundingSphere = new THREE.Sphere(new THREE.Vector3(), 2)

        const debris = new THREE.Points(geometry, new THREE.ShaderMaterial({
            uniforms: {
                centre: {value: new THREE.Vector3(0, 0, 1)},
                start: {value: NEVER},
                radius: {value: 0},
                time: uniforms.time,
                pixelsPerRadian,
                water: {value: 0},
            },
            vertexShader: debrisVertex,
            fragmentShader: debrisFragment,
            blending: THREE.AdditiveBlending,
            depthWrite: false,
            transparent: true,
        }))
        debris.visible = false

        const incoming = createRing(ringGeometry)

        group.add(flash, debris, incoming)
        return {centre: new THREE.Vector3(0, 0, 1), startedAt: NEVER, radius: 0, splash: false, flash, debris, incoming}
    })

    return {
        object: group,

        start(centre, radius, seconds, splash = false) {
            const index = oldest(slots)
            const slot = slots[index]

            slot.centre.copy(centre).normalize()
            slot.startedAt = seconds
            slot.radius = radius
            slot.splash = splash

            // The sea has no tiles to throw about or burn.
            uniforms.blasts.value[index].set(slot.centre.x, slot.centre.y, slot.centre.z, seconds)
            uniforms.blastRadii.value[index] = splash ? 0 : radius

            const material = slot.debris.material as THREE.ShaderMaterial
            material.uniforms.centre.value.copy(slot.centre)
            material.uniforms.start.value = seconds + BLAST_TIMELINE.fall
            material.uniforms.radius.value = splash ? radius * SPLASH_SCALE : radius
            material.uniforms.water.value = splash ? 1 : 0

            slot.flash.position.copy(slot.centre).multiplyScalar(1.01)
            slot.flash.material.color.set(splash ? SPLASH_FLASH : BLAST_FLASH)
            layRing(slot.incoming, slot.centre, radius * INCOMING_REACH)
            ringColour(slot.incoming, TARGET_RED)
        },

        setAim(centre, radius) {
            aim.visible = centre !== undefined
            if (centre) layRing(aim, centre, radius)
        },

        setCharge(progress) {
            (aim.material as THREE.ShaderMaterial).uniforms.progress.value = progress
        },

        newest(seconds) {
            let best: Slot | undefined
            for (const slot of slots) {
                const elapsed = seconds - slot.startedAt
                if (elapsed < 0 || elapsed > BLAST_TIMELINE.fall + BLAST_TIMELINE.shock + 1) continue
                if (!best || slot.startedAt > best.startedAt) best = slot
            }
            return best?.centre
        },

        update(seconds, camera) {
            uniforms.time.value = seconds
            camera.getWorldDirection(toCamera).negate()

            // A slow breath rather than a blink: this is held for many seconds.
            ringOpacity(aim, 0.85 + 0.15 * Math.sin(seconds * 4))

            slots.forEach((slot, index) => {
                const elapsed = seconds - slot.startedAt

                if (blastOver(elapsed)) {
                    // Freed, so the shader skips it on the cheapest test it has.
                    uniforms.blastRadii.value[index] = 0
                    slot.flash.visible = false
                    slot.debris.visible = false
                    slot.incoming.visible = false
                    return
                }

                const sinceImpact = elapsed - BLAST_TIMELINE.fall

                // The same ring closes in on every target; in the sea it then
                // opens back out as the ripple.
                const ring = slot.splash && sinceImpact >= 0 ? rippleAt(sinceImpact) : incomingAt(elapsed)
                slot.incoming.visible = ring.reach > 0.02 && ring.opacity > 0.01
                if (slot.incoming.visible) {
                    if (slot.splash && sinceImpact >= 0) ringColour(slot.incoming, SPLASH_RING)
                    slot.incoming.scale.setScalar(slot.radius * ring.reach * RING_MARGIN)
                    ringOpacity(slot.incoming, ring.opacity)
                }

                const hidden = isBehindGlobe(slot.flash.position, toCamera)

                slot.debris.visible = !hidden && !reducedMotion && sinceImpact >= 0 && sinceImpact < 1.6

                const {scale, opacity} = flashAt(elapsed, slot.radius)
                slot.flash.visible = !hidden && opacity > 0.01
                slot.flash.scale.setScalar(slot.splash ? scale * SPLASH_SCALE : scale)
                slot.flash.material.opacity = opacity
            })
        },

        dispose() {
            flashTexture.dispose()
            ringGeometry.dispose()
            ;(aim.material as THREE.Material).dispose()
            for (const slot of slots) {
                ;(slot.incoming.material as THREE.Material).dispose()
                slot.flash.material.dispose()
                slot.debris.geometry.dispose()
                ;(slot.debris.material as THREE.Material).dispose()
            }
            group.removeFromParent()
        },
    }
}

/**
 * The flash sprite over one blast: a red glow swelling while the bomb falls,
 * then a burst many times the crater's size that fades fast.
 */
export function flashAt(elapsed: number, radius: number): {scale: number, opacity: number} {
    if (elapsed < 0) return {scale: 0, opacity: 0}

    if (elapsed < BLAST_TIMELINE.fall) {
        const k = elapsed / BLAST_TIMELINE.fall
        return {scale: radius * (0.5 + 1.5 * k), opacity: 0.25 + 0.5 * k}
    }

    const s = elapsed - BLAST_TIMELINE.fall
    if (s > FLASH_SECONDS) return {scale: 0, opacity: 0}

    const k = s / FLASH_SECONDS
    return {scale: radius * (3 + 7 * (1 - (1 - k) ** 3)), opacity: (1 - k) ** 2}
}

/** The ripple a bomb leaves in the sea, in blast radii, after it lands. */
export function rippleAt(sinceImpact: number): {reach: number, opacity: number} {
    if (sinceImpact < 0 || sinceImpact >= RIPPLE_SECONDS) return {reach: 0, opacity: 0}

    const k = sinceImpact / RIPPLE_SECONDS
    return {reach: 0.3 + 2.2 * (1 - (1 - k) ** 2), opacity: (1 - k) ** 1.5}
}

/**
 * The ring closing in on a target while its bomb falls, in blast radii: wide
 * and faint when dropped, on the target and bright as it lands, gone after.
 */
export function incomingAt(elapsed: number): {reach: number, opacity: number} {
    if (elapsed < 0 || elapsed >= BLAST_TIMELINE.fall) return {reach: 0, opacity: 0}

    const k = elapsed / BLAST_TIMELINE.fall
    return {reach: INCOMING_REACH * (1 - k * k), opacity: 0.4 + 0.6 * k}
}

function createRing(geometry: THREE.PlaneGeometry): THREE.Mesh {
    const ring = new THREE.Mesh(geometry, new THREE.ShaderMaterial({
        uniforms: {
            colour: {value: TARGET_RED.clone()},
            opacity: {value: 1},
            edgeAt: {value: 1 / RING_MARGIN},
            progress: {value: 0},
        },
        vertexShader: ringVertex,
        fragmentShader: ringFragment,
        transparent: true,
        depthWrite: false,
    }))
    ring.visible = false
    // After the tiles, so it is laid over them rather than sorted under.
    ring.renderOrder = 2
    return ring
}

/**
 * Lays a ring flat on the globe at `centre`, `radius` radians wide. Flat is
 * close enough at these sizes: the sphere falls away under a 0.15 rad ring by
 * about a hundredth of the radius, and the lift keeps it clear of the tiles.
 */
function layRing(ring: THREE.Mesh, centre: THREE.Vector3, radius: number) {
    const normal = centre.clone().normalize()
    ring.position.copy(normal).multiplyScalar(1.003)
    ring.quaternion.setFromUnitVectors(Z_AXIS, normal)
    ring.scale.setScalar(radius * RING_MARGIN)
}

function ringColour(ring: THREE.Mesh, colour: THREE.Color) {
    (ring.material as THREE.ShaderMaterial).uniforms.colour.value.copy(colour)
}

function ringOpacity(ring: THREE.Mesh, opacity: number) {
    (ring.material as THREE.ShaderMaterial).uniforms.opacity.value = opacity
}

function oldest(slots: Slot[]): number {
    let index = 0
    for (let i = 1; i < slots.length; i++) {
        if (slots[i].startedAt < slots[index].startedAt) index = i
    }
    return index
}

/** One random heading, reach and throw per spark, shared by every slot. */
function debrisSeeds(): THREE.BufferAttribute {
    const values = new Float32Array(DEBRIS_PER_BLAST * 3)
    for (let i = 0; i < DEBRIS_PER_BLAST; i++) {
        values[i * 3] = Math.random() * Math.PI * 2
        values[i * 3 + 1] = 0.3 + Math.random() * 0.7
        values[i * 3 + 2] = Math.random()
    }
    return new THREE.BufferAttribute(values, 3)
}

function glowTexture(): THREE.Texture {
    const size = 128
    const canvas = document.createElement("canvas")
    canvas.width = canvas.height = size

    const context = canvas.getContext("2d")
    if (context) {
        const gradient = context.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2)
        gradient.addColorStop(0, "rgba(255,255,240,1)")
        gradient.addColorStop(0.2, "rgba(255,210,120,0.9)")
        gradient.addColorStop(0.5, "rgba(255,90,20,0.35)")
        gradient.addColorStop(1, "rgba(255,40,0,0)")
        context.fillStyle = gradient
        context.fillRect(0, 0, size, size)
    }

    return new THREE.CanvasTexture(canvas)
}
