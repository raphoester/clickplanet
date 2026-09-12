import * as THREE from "three"
import {isBehindGlobe} from "./bonusBox.ts"
import "./bonusPointer.css"

/** How far in from the edge the pointer sits, as a share of the half-viewport. */
const MARGIN = 0.12

export type PointerPlacement = {
    /** Normalised device coordinates, clamped to the inset edge. */
    x: number
    y: number
    /** Degrees clockwise from "pointing right", for a CSS rotate. */
    angle: number
}

/** Whether the box is outside the frame, which is the only time to point at it. */
export function offScreen(ndc: {x: number, y: number}): boolean {
    return Math.abs(ndc.x) > 1 || Math.abs(ndc.y) > 1
}

/**
 * Where to put the pointer for a box at `ndc`: on the inset edge, along the ray
 * from the middle of the screen, turned to face it.
 */
export function placePointer(ndc: {x: number, y: number}, margin = MARGIN): PointerPlacement {
    const limit = 1 - margin

    // The ray hits whichever edge it runs out of room on first.
    const reach = Math.min(
        Math.abs(ndc.x) > 1e-6 ? limit / Math.abs(ndc.x) : Infinity,
        Math.abs(ndc.y) > 1e-6 ? limit / Math.abs(ndc.y) : Infinity,
    )

    const scale = Number.isFinite(reach) ? reach : 0

    return {
        x: ndc.x * scale,
        y: ndc.y * scale,
        // Screen y grows downward where NDC y grows up, hence the negation.
        angle: Math.atan2(-ndc.y, ndc.x) * (180 / Math.PI),
    }
}

export type BonusPointer = {
    /** Points at `position`, or hides when it is undefined, on screen, or hidden. */
    update(position: THREE.Vector3 | undefined, camera: THREE.OrthographicCamera): void
    dispose(): void
}

/**
 * An arrow at the edge of the screen saying which way the box is.
 *
 * Zoomed in, the box is almost always outside the frame — the orbit is in world
 * units and the visible world shrinks — so without this a zoomed player never
 * learns a box was theirs until it has gone.
 *
 * Written to the DOM rather than into the scene, and driven from the animation
 * loop rather than from React: it moves every frame, and this sits beside a
 * WebGL scene that wants the main thread.
 */
export function createBonusPointer(container: HTMLElement, label = "?", variant?: string): BonusPointer {
    const root = document.createElement("div")
    root.className = variant ? `bonus-pointer bonus-pointer--${variant}` : "bonus-pointer"
    root.hidden = true

    const arrow = document.createElement("span")
    arrow.className = "bonus-pointer-arrow"

    const badge = document.createElement("span")
    badge.className = "bonus-pointer-badge"
    badge.textContent = label

    root.append(arrow, badge)
    container.append(root)

    const projected = new THREE.Vector3()
    const toCamera = new THREE.Vector3()

    return {
        update(position, camera) {
            if (!position) {
                root.hidden = true
                return
            }

            camera.getWorldDirection(toCamera).negate()

            // A box on the far side is not somewhere the player can go and take
            // it, so pointing at it would send them the wrong way.
            if (isBehindGlobe(position, toCamera)) {
                root.hidden = true
                return
            }

            const ndc = projected.copy(position).project(camera)
            if (!offScreen(ndc)) {
                root.hidden = true
                return
            }

            const {x, y, angle} = placePointer(ndc)

            root.hidden = false
            root.style.left = `${(x * 0.5 + 0.5) * 100}%`
            root.style.top = `${(-y * 0.5 + 0.5) * 100}%`
            arrow.style.setProperty("--bonus-pointer-angle", `${angle}deg`)
        },

        dispose() {
            root.remove()
        },
    }
}
