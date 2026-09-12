import * as THREE from "three"

// The same two shaders the atmosphere's limb glow is drawn with. The shader is
// the part worth sharing — a Fresnel term over a back-facing shell — while the
// radius, the colour and the blend flags are this halo's own.
import glowVertex from "./shaders/atmosphere/vertex.glsl"
import glowFragment from "./shaders/atmosphere/fragment.glsl"

import {mulberry32} from "./stars.ts"

/** Outside the 1.0 tile shell, so the box never sinks into the flags. */
const ORBIT_RADIUS = 1.15

/**
 * The box's size in world units, like every other object in the scene.
 *
 * It is deliberately *not* held at a constant apparent size. Under an
 * orthographic camera `zoom` scales the whole world, so an object that divides
 * it out stays the same number of pixels while the ground beneath it grows —
 * which does not read as a rule, it reads as a bug: the box looks pinned to the
 * screen rather than flying over the planet.
 */
const WORLD_SIZE = 0.1

/**
 * A guard, not a behaviour: the visible world is 2/zoom tall, so past about 9x
 * a world-sized box would take over the screen. It only ever bites at a zoom the
 * orbit is barely ever in view at.
 */
const MAX_APPARENT_SIZE = 0.9

const SPIN_PER_SECOND = 0.8

/** A full orbit in about sixteen seconds: at six, players could not land a click. */
const ORBIT_PER_SECOND = 0.4

/** How long a box is up at most; the offer's own deadline can cut it shorter. */
const LIFETIME_SECONDS = 12

/** The tail of that life it spends fading, so it is never cut off mid-flight. */
const FADE_SECONDS = 0.9

/** The pop when it is taken: it swells and fades where it was caught. */
const BURST_SECONDS = 0.45
const BURST_SCALE = 2.2
const BURST_GLOW = 3

/**
 * The halo, in the box's own units: the box is one unit across, so this shell
 * sits just outside its corners at 0.87.
 *
 * It has to hug the silhouette. This glow is brightest at the *middle* of the
 * shell and falls to nothing at its edge — the atmosphere reads as a rim only
 * because the earth is opaque and fills 94% of it. A shell much wider than what
 * it wraps leaves that bright middle on show, which is a lens-flare blob sitting
 * over the map rather than a glow around a box.
 */
const GLOW_RADIUS = 1.25
const GLOW_POWER = 2.6
const GLOW_INTENSITY = 1.4

/**
 * Written straight to the framebuffer like the atmosphere's, so read these as
 * the colour on screen. Warm, against the blue limb and the black sky.
 */
const GLOW_COLOUR = new THREE.Color(1.0, 0.68, 0.16)

/**
 * The key light, painted on rather than lit.
 *
 * There is no light in this scene but a flat ambient one, so a shaded material
 * would render as a single tone and the box would read as a sticker. These are
 * the six faces' brightnesses in `BoxGeometry`'s material order — +X, -X, +Y,
 * -Y, +Z, -Z — so the top face catches a sun that does not exist, the four sides
 * sit mid, and the underside falls away. Rotation is what turns that into a
 * solid: the silhouette changes, and the tone changes with the face.
 */
export const FACE_TONES = [0.74, 0.74, 1, 0.4, 0.74, 0.74]

const TEXTURE_SIZE = 128

/** The plane the box travels in, and where on it the box starts. */
export type Orbit = {
    /** Two orthonormal directions spanning the plane. */
    u: THREE.Vector3
    v: THREE.Vector3
    /** Where on the circle the box begins, in radians. */
    phase: number
}

export type BonusBox = {
    /** Add this to the scene once; it carries the box and its halo. */
    readonly object: THREE.Object3D

    /** True while a box is up, whether flying or playing its pop. */
    readonly visible: boolean

    /** True only while it is still there to be caught. */
    readonly flying: boolean

    /**
     * Puts a box on a fresh orbit drawn from `seed`.
     *
     * `deadline` is when it must be gone, on the clock `update` is given. The
     * lifetime alone is not enough: it starts at the first frame drawn, and the
     * server's token started lapsing when the offer was sent — a slow stream or
     * a hidden tab (which draws no frames) spends that difference, and a box
     * caught after the token lapsed pops and wins nothing.
     */
    spawn(seed: number, deadline?: number): void

    /**
     * Takes the box, starting the pop where it was caught. False when there was
     * nothing to take — already caught, or already gone.
     */
    take(): boolean

    hide(): void

    /** `seconds` is a clock that only goes forward, not a delta. */
    update(seconds: number, camera: THREE.OrthographicCamera): void

    /**
     * Whether a click at `ndc` — normalised device coordinates — lands on the
     * box. False while it is gone, already taken, or behind the planet, so the
     * click falls through to the tile under it.
     */
    hitTest(camera: THREE.OrthographicCamera, ndc: THREE.Vector2): boolean

    dispose(): void
}

/**
 * A question-mark box flying past the planet, and the click that takes it.
 *
 * Deliberately unlit: every material here is `MeshBasicMaterial` or a raw
 * shader, so nothing in it depends on a light the scene does not have, and
 * adding one later cannot change how it looks. What stands in for shading is a
 * painted key light per face (see `FACE_TONES`), a dark border baked into the
 * texture to hold the silhouette against the map, and the spin.
 */
export function createBonusBox(): BonusBox {
    const texture = faceTexture()

    const geometry = new THREE.BoxGeometry(1, 1, 1)
    const materials = FACE_TONES.map((tone) => new THREE.MeshBasicMaterial({
        map: texture,
        color: new THREE.Color(tone, tone, tone),
        transparent: true,
    }))

    const box = new THREE.Mesh(geometry, materials)

    const glowGeometry = new THREE.IcosahedronGeometry(GLOW_RADIUS, 8)
    const glowMaterial = new THREE.ShaderMaterial({
        uniforms: {
            colour: {value: GLOW_COLOUR},
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

    // The box has to be drawn before its halo. It is transparent, so that it can
    // fade when taken, and that puts it in the same pass as the shell — where
    // three sorts on distance, and these two share a centre. Left to that sort
    // the shell can win, and its bright middle is then never cut away by the
    // box's depth: the halo becomes the blob it is shaped to avoid.
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
            if (phase === "gone") return

            // The first frame after a spawn is what the clock is measured from,
            // so a box always appears where its orbit begins.
            if (spawnedAt === undefined) {
                spawnedAt = seconds
                lifetime = Math.min(LIFETIME_SECONDS, deadline - seconds)
            }

            const scale = boxScale(camera.zoom)

            if (phase === "taken") {
                // The position is left where it was caught, so the pop happens
                // under the cursor that earned it rather than flying away.
                if (takenAt === undefined) takenAt = seconds

                const pop = burstAt(seconds - takenAt)
                group.scale.setScalar(scale * pop.scale)
                setFade(pop.opacity, BURST_GLOW)

                if (pop.opacity <= 0) stop()
                return
            }

            const age = seconds - spawnedAt

            group.position.copy(orbitPosition(orbit, age * ORBIT_PER_SECOND))
            group.scale.setScalar(scale)
            box.rotation.set(age * SPIN_PER_SECOND, age * SPIN_PER_SECOND * 0.7, 0)

            const opacity = flightOpacity(age, lifetime)
            setFade(opacity, 1)

            if (opacity <= 0) stop()
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
            texture.dispose()
        },
    }
}

/**
 * The box's scale at a given zoom: its world size, until that would take over
 * the screen.
 */
export function boxScale(zoom: number): number {
    return Math.min(WORLD_SIZE, MAX_APPARENT_SIZE / zoom)
}

/** Full until the last `fade` seconds of the box's life, then out. */
export function flightOpacity(
    age: number,
    lifetime = LIFETIME_SECONDS,
    fade = FADE_SECONDS,
): number {
    const left = lifetime - age
    if (left <= 0) return 0

    return Math.min(1, left / fade)
}

/** The pop a taken box plays: out from where it was caught, and away. */
export function burstAt(since: number, duration = BURST_SECONDS): {scale: number, opacity: number} {
    const through = Math.min(1, Math.max(0, since / duration))

    return {
        // Fast out then easing to a stop, the way a struck thing moves.
        scale: 1 + (BURST_SCALE - 1) * (1 - (1 - through) * (1 - through)),
        opacity: 1 - through,
    }
}

/**
 * An orbit drawn from a seed, so the server can later name the one every client
 * has to draw by sending a number rather than a path.
 *
 * The axis is spread evenly over the sphere the same way the sky is: equal
 * height is equal area, so no orbit is favoured. Drawing a latitude instead
 * would crowd the orbits around one plane.
 */
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

    // Any direction not parallel to the axis will do to get the first in-plane
    // vector; the second is then the one perpendicular to both.
    const off = Math.abs(axis.y) > 0.9
        ? new THREE.Vector3(1, 0, 0)
        : new THREE.Vector3(0, 1, 0)

    const u = new THREE.Vector3().crossVectors(axis, off).normalize()
    const v = new THREE.Vector3().crossVectors(axis, u).normalize()

    return {u, v, phase: 2 * Math.PI * random()}
}

/** Where on its orbit the box is, `angle` radians in. */
export function orbitPosition(orbit: Orbit, angle: number, radius = ORBIT_RADIUS): THREE.Vector3 {
    const theta = orbit.phase + angle

    return new THREE.Vector3()
        .addScaledVector(orbit.u, Math.cos(theta) * radius)
        .addScaledVector(orbit.v, Math.sin(theta) * radius)
}

/**
 * Whether the planet is in front of a point, for an orthographic camera.
 *
 * The depth test already hides the box itself, but a ray cast at one object
 * knows nothing about the globe and would report a hit straight through it. With
 * a box frustum the test is exact and needs no ray: a point is hidden when it is
 * on the far side of the centre and lies within the globe's radius of the view
 * axis.
 *
 * `toCamera` must be a unit vector pointing from the scene towards the camera.
 */
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

/**
 * The `?` face, drawn rather than loaded.
 *
 * A canvas keeps this out of the content-addressed asset pipeline — there is no
 * file to hash, deploy or cache-bust for one 128px square, and nothing to load
 * before the box can first appear.
 *
 * The dark border is what makes the box readable at the ~40px it occupies: over
 * a map of saturated flags, the silhouette is most of what the eye gets, and an
 * unbordered gold square dissolves into a yellow country.
 */
function faceTexture(): THREE.CanvasTexture {
    const canvas = document.createElement("canvas")
    canvas.width = TEXTURE_SIZE
    canvas.height = TEXTURE_SIZE

    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the bonus box face")

    const size = TEXTURE_SIZE

    context.fillStyle = "#f2a91c"
    context.fillRect(0, 0, size, size)

    context.strokeStyle = "#4a2c02"
    context.lineWidth = size * 0.09
    context.strokeRect(context.lineWidth / 2, context.lineWidth / 2, size - context.lineWidth, size - context.lineWidth)

    // The rivets are Mario's, and they also break up the flat ground so the
    // face reads as a panel rather than as a swatch.
    context.fillStyle = "#4a2c02"
    const inset = size * 0.19
    const rivet = size * 0.035
    for (const [x, y] of [[inset, inset], [size - inset, inset], [inset, size - inset], [size - inset, size - inset]]) {
        context.beginPath()
        context.arc(x, y, rivet, 0, 2 * Math.PI)
        context.fill()
    }

    context.font = `bold ${size * 0.62}px system-ui, -apple-system, "Segoe UI", sans-serif`
    context.textAlign = "center"
    context.textBaseline = "middle"

    context.fillStyle = "#4a2c02"
    context.fillText("?", size / 2, size * 0.54)
    context.fillStyle = "#fff6df"
    context.fillText("?", size / 2, size * 0.51)

    const texture = new THREE.CanvasTexture(canvas)
    // Without this the gold is read as a linear colour and comes out washed out.
    texture.colorSpace = THREE.SRGBColorSpace

    return texture
}
