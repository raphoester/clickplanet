import * as THREE from "three"
import {isBehindGlobe} from "./bonusBox.ts"
import {blastMarkSvg} from "./blastMark.ts"
import {questionMarkSvg} from "./questionMark.ts"
import "./bonusPointer.css"

const MARGIN = 0.12

export type PointerPlacement = {
    x: number
    y: number
    angle: number
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

export type BonusPointer = {
    update(position: THREE.Vector3 | undefined, camera: THREE.OrthographicCamera): void
    dispose(): void
}

export function createBonusPointer(container: HTMLElement, variant?: "blast"): BonusPointer {
    const root = document.createElement("div")
    root.className = variant ? `bonus-pointer bonus-pointer--${variant}` : "bonus-pointer"
    root.hidden = true

    const arrow = document.createElement("span")
    arrow.className = "bonus-pointer-arrow"

    const badge = document.createElement("span")
    badge.className = "bonus-pointer-badge"
    badge.append(variant === "blast"
        ? blastMarkSvg("bonus-pointer-mark")
        : questionMarkSvg("bonus-pointer-mark", "var(--cream)"))

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
