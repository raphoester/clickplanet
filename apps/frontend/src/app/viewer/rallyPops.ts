import * as THREE from "three"
import {offScreen} from "./bonusPointer.ts"
import "./rallyPops.css"

// More than this at once and the oldest goes first: a fast thumb makes no pile.
const MAX_POPS = 8

// Gone after its animation; the timer is for a page that runs none (reduced motion, a hidden tab).
const POP_MS = 900

export type RallyPops = {
    // A click a rally made cheaper: a short "2×" rises from its tile.
    pop(position: THREE.Vector3, camera: THREE.Camera, text: string): void
    dispose(): void
}

export function createRallyPops(container: HTMLElement): RallyPops {
    const shown: HTMLElement[] = []
    const projected = new THREE.Vector3()

    const remove = (pop: HTMLElement) => {
        pop.remove()
        const index = shown.indexOf(pop)
        if (index >= 0) shown.splice(index, 1)
    }

    return {
        pop(position, camera, text) {
            const ndc = projected.copy(position).project(camera)
            if (offScreen(ndc)) return

            if (shown.length >= MAX_POPS) remove(shown[0])

            const pop = document.createElement("span")
            pop.className = "rally-pop"
            pop.setAttribute("aria-hidden", "true")
            pop.textContent = text
            pop.style.left = `${((ndc.x * 0.5 + 0.5) * 100).toFixed(3)}%`
            pop.style.top = `${((-ndc.y * 0.5 + 0.5) * 100).toFixed(3)}%`
            pop.addEventListener("animationend", () => remove(pop), {once: true})
            setTimeout(() => remove(pop), POP_MS)

            container.append(pop)
            shown.push(pop)
        },

        dispose() {
            for (const pop of [...shown]) remove(pop)
        },
    }
}
