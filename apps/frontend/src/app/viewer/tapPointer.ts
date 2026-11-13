import * as THREE from "three"
import {isBehindGlobe} from "./bonusBox.ts"
import {offScreen} from "./bonusPointer.ts"
import "./tapPointer.css"

const SVG_NS = "http://www.w3.org/2000/svg"

// A glove with its index finger up; `fingertip` is the point that lands on the tile.
export const HAND = {
    width: 44,
    height: 58,
    fingertip: {x: 14, y: 3},
    shapes: [
        {x: 8.5, y: 3, width: 11, height: 31, r: 5.5},
        {x: 8.5, y: 23, width: 30, height: 25, r: 9},
        {x: 18.5, y: 19.5, width: 9.5, height: 15, r: 4.75},
        {x: 27, y: 21.5, width: 9.5, height: 14, r: 4.75},
        {x: 13, y: 45, width: 22, height: 10, r: 3},
    ],
    thumb: "M 10 31 C 3 30, 1 37, 6 40 L 14 44 C 18 46, 21 41, 18 38 Z",
    creases: ["M 19.2 21.5 L 19.2 32", "M 27.6 23.5 L 27.6 32.5", "M 14 45.5 L 34 45.5"],
    outline: 5,
    crease: 2,
} as const

// Above the fingertip, clear of the hand below it.
const BURST_ANGLES = [-150, -90, -30]

export type TapPointer = {
    update(position: THREE.Vector3 | undefined, camera: THREE.OrthographicCamera): void
    dispose(): void
}

export function createTapPointer(container: HTMLElement): TapPointer {
    const root = document.createElement("div")
    root.className = "tap-pointer"
    root.hidden = true

    const hand = handSvg()
    root.append(burstSvg(), hand)
    container.append(root)

    const projected = new THREE.Vector3()
    const toCamera = new THREE.Vector3()
    let left = ""
    let top = ""

    const hide = () => {
        if (!root.hidden) root.hidden = true
    }

    return {
        update(position, camera) {
            if (!position) {
                hide()
                return
            }

            camera.getWorldDirection(toCamera).negate()
            const ndc = projected.copy(position).project(camera)
            if (isBehindGlobe(position, toCamera) || offScreen(ndc)) {
                hide()
                return
            }

            const x = `${(ndc.x * 0.5 + 0.5) * 100}%`
            const y = `${(-ndc.y * 0.5 + 0.5) * 100}%`
            if (x !== left) root.style.left = left = x
            if (y !== top) root.style.top = top = y
            if (root.hidden) root.hidden = false
        },

        dispose() {
            root.remove()
        },
    }
}

function handSvg(): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", `0 0 ${HAND.width} ${HAND.height}`)
    svg.setAttribute("class", "tap-pointer-hand")
    svg.setAttribute("aria-hidden", "true")
    svg.style.width = `${HAND.width}px`
    svg.style.height = `${HAND.height}px`
    svg.style.setProperty("--tap-tip-x", `${HAND.fingertip.x}px`)
    svg.style.setProperty("--tap-tip-y", `${HAND.fingertip.y}px`)

    const outline = gloveGroup("tap-pointer-outline")
    outline.setAttribute("stroke-width", String(HAND.outline))
    const glove = gloveGroup("tap-pointer-glove")

    const creases = document.createElementNS(SVG_NS, "g")
    creases.setAttribute("class", "tap-pointer-creases")
    creases.setAttribute("stroke-width", String(HAND.crease))
    for (const d of HAND.creases) creases.append(pathOf(d))

    svg.append(outline, glove, creases)
    return svg
}

function gloveGroup(className: string): SVGGElement {
    const group = document.createElementNS(SVG_NS, "g")
    group.setAttribute("class", className)
    for (const {x, y, width, height, r} of HAND.shapes) {
        const rect = document.createElementNS(SVG_NS, "rect")
        rect.setAttribute("x", String(x))
        rect.setAttribute("y", String(y))
        rect.setAttribute("width", String(width))
        rect.setAttribute("height", String(height))
        rect.setAttribute("rx", String(r))
        group.append(rect)
    }
    group.append(pathOf(HAND.thumb))
    return group
}

function burstSvg(): SVGSVGElement {
    const svg = document.createElementNS(SVG_NS, "svg")
    svg.setAttribute("viewBox", "-20 -20 40 40")
    svg.setAttribute("class", "tap-pointer-burst")
    svg.setAttribute("aria-hidden", "true")
    for (const className of ["tap-pointer-burst-ink", "tap-pointer-burst-light"]) {
        const group = document.createElementNS(SVG_NS, "g")
        group.setAttribute("class", className)
        for (const angle of BURST_ANGLES) {
            const radians = angle * Math.PI / 180
            const line = document.createElementNS(SVG_NS, "line")
            line.setAttribute("x1", (Math.cos(radians) * 10).toFixed(2))
            line.setAttribute("y1", (Math.sin(radians) * 10).toFixed(2))
            line.setAttribute("x2", (Math.cos(radians) * 16).toFixed(2))
            line.setAttribute("y2", (Math.sin(radians) * 16).toFixed(2))
            group.append(line)
        }
        svg.append(group)
    }
    return svg
}

function pathOf(d: string): SVGPathElement {
    const path = document.createElementNS(SVG_NS, "path")
    path.setAttribute("d", d)
    return path
}
