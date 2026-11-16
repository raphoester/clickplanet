import * as THREE from "three"
import {isBehindGlobe} from "./bonusBox.ts"
import {blastMarkSvg} from "./blastMark.ts"
import {rallyMarkSvg} from "./rallyMark.ts"
import {questionMarkSvg} from "./questionMark.ts"
import "./bonusPointer.css"

const MARGIN = 0.12

// How far past the globe's edge the arrow to a point round its back sits, in CSS pixels: its badge clears the edge.
const RIM_GAP = 22

export type PointerPlacement = {
    x: number
    y: number
    angle: number
}

// The canvas in CSS pixels.
export type ViewSize = {
    width: number
    height: number
}

// The globe on the screen: its centre in device coordinates, its radius in CSS pixels.
export type GlobeDisc = {
    x: number
    y: number
    radius: number
}

export function globeDiscOf(camera: THREE.Camera, size: ViewSize): GlobeDisc {
    const centre = new THREE.Vector3().project(camera)
    const top = new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 1).project(camera)
    return {
        x: centre.x,
        y: centre.y,
        radius: Math.hypot((top.x - centre.x) * size.width / 2, (top.y - centre.y) * size.height / 2),
    }
}

export function offScreen(ndc: {x: number, y: number}): boolean {
    return Math.abs(ndc.x) > 1 || Math.abs(ndc.y) > 1
}

export function placePointer(ndc: {x: number, y: number}, margin = MARGIN): PointerPlacement {
    const limit = 1 - margin

    const reach = Math.min(
        Math.abs(ndc.x) > 1e-6 ? limit / Math.abs(ndc.x) : Infinity,
        Math.abs(ndc.y) > 1e-6 ? limit / Math.abs(ndc.y) : Infinity,
    )

    const scale = Number.isFinite(reach) ? reach : 0

    return {
        x: ndc.x * scale,
        y: ndc.y * scale,
        angle: Math.atan2(-ndc.y, ndc.x) * (180 / Math.PI),
    }
}

// Where the arrow to a point round the back of the globe goes: just past the globe's edge on the point's side, facing
// out, held inside the screen when that edge is off it.
export function placeOnRim(point: {x: number, y: number}, disc: GlobeDisc, size: ViewSize, margin = MARGIN): PointerPlacement {
    const halfWidth = size.width / 2
    const halfHeight = size.height / 2
    if (halfWidth <= 0 || halfHeight <= 0) return placePointer(point, margin)

    const dx = (point.x - disc.x) * halfWidth
    const dy = (point.y - disc.y) * halfHeight
    const length = Math.hypot(dx, dy)
    const [ux, uy] = length > 1e-6 ? [dx / length, dy / length] : [0, 1]
    const reach = disc.radius + RIM_GAP
    const rim = {x: disc.x + ux * reach / halfWidth, y: disc.y + uy * reach / halfHeight}
    const angle = Math.atan2(-uy, ux) * (180 / Math.PI)

    const limit = 1 - margin
    if (Math.abs(rim.x) <= limit && Math.abs(rim.y) <= limit) return {...rim, angle}
    return {...placePointer(rim, margin), angle}
}

export type BonusPointer = {
    // A rally's arrow needs the canvas's size: it points round the back of the globe from its edge.
    update(position: THREE.Vector3 | undefined, camera: THREE.OrthographicCamera, size?: ViewSize): void
    dispose(): void
}

export function createBonusPointer(container: HTMLElement, variant?: "blast" | "rally"): BonusPointer {
    const root = document.createElement("div")
    root.className = variant ? `bonus-pointer bonus-pointer--${variant}` : "bonus-pointer"
    root.hidden = true

    const arrow = document.createElement("span")
    arrow.className = "bonus-pointer-arrow"

    const badge = document.createElement("span")
    badge.className = "bonus-pointer-badge"
    badge.append(variant === "blast"
        ? blastMarkSvg("bonus-pointer-mark")
        : variant === "rally"
            ? rallyMarkSvg("bonus-pointer-mark")
            : questionMarkSvg("bonus-pointer-mark", "var(--cream)"))

    root.append(arrow, badge)
    container.append(root)

    const projected = new THREE.Vector3()
    const toCamera = new THREE.Vector3()

    return {
        update(position, camera, size) {
            if (!position) {
                root.hidden = true
                return
            }

            camera.getWorldDirection(toCamera).negate()

            // A box behind the globe flies round to the front; a rally's ring stays where it is, so its arrow points round.
            const behind = isBehindGlobe(position, toCamera)
            if (behind && variant !== "rally") {
                root.hidden = true
                return
            }

            const ndc = projected.copy(position).project(camera)
            if (!behind && !offScreen(ndc)) {
                root.hidden = true
                return
            }

            const {x, y, angle} = behind && size ? placeOnRim(ndc, globeDiscOf(camera, size), size) : placePointer(ndc)

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
