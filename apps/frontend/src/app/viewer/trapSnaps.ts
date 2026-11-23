import * as THREE from "three"
import {offScreen} from "./bonusPointer.ts"
import "./trapSnaps.css"

// More than this at once and the oldest goes first: a war of traps makes no pile.
const MAX_SNAPS = 6

// Gone after its animation; the timer is for a page that runs none (reduced motion, a hidden tab).
const SNAP_MS = 1300

const SVG = "http://www.w3.org/2000/svg"

// A jaw is half a ring of steel with its teeth along the flat edge, drawn for the top; the bottom one is turned over.
const JAW = "M -42 0 A 42 42 0 0 1 42 0 L 22 0 A 22 22 0 0 0 -22 0 Z"
const TEETH = [-36, -27, -18, -9, 0, 9, 18, 27, 36]
    .map((x) => `M ${x - 4.5} 0 L ${x} 11 L ${x + 4.5} 0 Z`)
    .join(" ")

export type TrapSnaps = {
    // A trap sprang on the tile: steel jaws slam shut over it, it flashes and shakes, and what left the bank pops.
    snap(position: THREE.Vector3, camera: THREE.Camera, clicks: string | undefined): void
    dispose(): void
}

function jaw(side: "top" | "bottom"): SVGGElement {
    const group = document.createElementNS(SVG, "g")
    group.setAttribute("class", `trap-snap-jaw trap-snap-jaw--${side}`)
    const steel = document.createElementNS(SVG, "path")
    steel.setAttribute("class", "trap-snap-steel")
    steel.setAttribute("d", JAW)
    const teeth = document.createElementNS(SVG, "path")
    teeth.setAttribute("class", "trap-snap-teeth")
    teeth.setAttribute("d", TEETH)
    const turned = document.createElementNS(SVG, "g")
    if (side === "bottom") turned.setAttribute("transform", "scale(1 -1) translate(4.5 0)")
    turned.append(steel, teeth)
    group.append(turned)
    return group
}

export function createTrapSnaps(container: HTMLElement): TrapSnaps {
    const shown: HTMLElement[] = []
    const projected = new THREE.Vector3()

    const remove = (snap: HTMLElement) => {
        snap.remove()
        const index = shown.indexOf(snap)
        if (index >= 0) shown.splice(index, 1)
    }

    return {
        snap(position, camera, clicks) {
            const ndc = projected.copy(position).project(camera)
            if (offScreen(ndc)) return

            if (shown.length >= MAX_SNAPS) remove(shown[0])

            const snap = document.createElement("div")
            snap.className = "trap-snap"
            snap.setAttribute("aria-hidden", "true")
            snap.style.left = `${((ndc.x * 0.5 + 0.5) * 100).toFixed(3)}%`
            snap.style.top = `${((-ndc.y * 0.5 + 0.5) * 100).toFixed(3)}%`

            const flash = document.createElement("span")
            flash.className = "trap-snap-flash"

            const jaws = document.createElementNS(SVG, "svg")
            jaws.setAttribute("class", "trap-snap-jaws")
            jaws.setAttribute("viewBox", "-50 -50 100 100")
            jaws.append(jaw("top"), jaw("bottom"))

            snap.append(flash, jaws)
            if (clicks) {
                const pop = document.createElement("span")
                pop.className = "trap-snap-pop"
                pop.textContent = clicks
                snap.append(pop)
            }

            snap.addEventListener("animationend", (event) => {
                if (event.target === snap && event.animationName === "trap-snap-fade") remove(snap)
            })
            setTimeout(() => remove(snap), SNAP_MS)

            container.append(snap)
            shown.push(snap)
        },

        dispose() {
            for (const snap of [...shown]) remove(snap)
        },
    }
}
