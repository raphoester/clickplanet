import * as THREE from "three"
import {Rally} from "../../domain/rallies.ts"
import {Region, regions} from "./atlas.ts"
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts"
import {createBonusPointer, offScreen} from "./bonusPointer.ts"
import {isBehindGlobe} from "./bonusBox.ts"
import haloVertex from "./shaders/rallyHalo/vertex.glsl"
import haloFragment from "./shaders/rallyHalo/fragment.glsl"
import "./rallyMarks.css"

// More rallies than this at once and the oldest of the other flags' go unmarked: the page's own come first.
const MAX_RALLIES = 24

// The halo spreads a little past the ring's edge.
const HALO_MARGIN = 1.3

// --rally for every rally: the flag planted at its centre says whose. Another flag's is fainter.
const RALLY = new THREE.Color(0x0fa3b1)

const OWN_OPACITY = 1
const OTHER_OPACITY = 0.55

const Z_AXIS = new THREE.Vector3(0, 0, 1)

// The cloth of the planted flag, in CSS pixels.
export const CLOTH = {width: 40, height: 27} as const

// Who called a rally, by account: its name as the page may show it, and the style that paints it in its color.
export type RallyCallers = {
    ask(accountId: string): void
    nameplateOf(accountId: string): {name: string, style: Record<string, string | number>} | undefined
}

export type RallyMarks = {
    readonly object: THREE.Object3D

    // Every rally on: a halo over its ring, its flag planted at its centre with its caller's name, the page's flag's in
    // the rally's colour.
    show(rallies: readonly Rally[], flag: string, positions: ArrayLike<number>): void

    // Each mark goes at its rally's end; each flag follows the camera; the arrow points at the newest ring of the page's
    // flag off the screen. True when a halo went, so the frame is drawn again.
    update(now: number, camera: THREE.OrthographicCamera): boolean

    dispose(): void
}

type Marked = {
    rally: Rally
    own: boolean
    centre: THREE.Vector3
    pin: HTMLElement
    name: HTMLElement
    shownName?: string
    left?: string
    top?: string
}

export function createRallyMarks(container: HTMLElement, callers: RallyCallers | undefined): RallyMarks {
    const group = new THREE.Group()
    const geometry = new THREE.PlaneGeometry(2, 2)
    const halos = Array.from({length: MAX_RALLIES}, () => {
        const halo = createHalo(geometry)
        group.add(halo)
        return halo
    })
    const pointer = createBonusPointer(container, "rally")
    const projected = new THREE.Vector3()
    const toCamera = new THREE.Vector3()

    let marked: Marked[] = []

    const unpin = () => {
        for (const each of marked) each.pin.remove()
        marked = []
    }

    return {
        object: group,

        show(rallies, flag, positions) {
            unpin()
            marked = [...rallies]
                .sort((a, b) => Number(b.countryId === flag) - Number(a.countryId === flag) || b.endsAt - a.endsAt)
                .slice(0, MAX_RALLIES)
                .map((rally) => {
                    const at = (rally.tile - 1) * 3
                    const centre = new THREE.Vector3(positions[at], positions[at + 1], positions[at + 2])
                    const own = rally.countryId === flag
                    const {pin, name} = createPin(rally.countryId, own)
                    container.append(pin)
                    callers?.ask(rally.accountId)
                    return {rally, own, centre, pin, name}
                })

            halos.forEach((halo, index) => {
                const each = marked[index]
                halo.visible = each !== undefined
                if (!each) return
                layHalo(halo, each.centre, each.rally.radius)
                const uniforms = (halo.material as THREE.ShaderMaterial).uniforms
                uniforms.colour.value.copy(RALLY)
                uniforms.opacity.value = each.own ? OWN_OPACITY : OTHER_OPACITY
            })
        },

        update(now, camera) {
            camera.getWorldDirection(toCamera).negate()

            let changed = false
            marked.forEach((each, index) => {
                const on = each.rally.endsAt > now
                if (halos[index].visible !== on) {
                    halos[index].visible = on
                    changed = true
                }

                const ndc = projected.copy(each.centre).project(camera)
                each.pin.hidden = !on || isBehindGlobe(each.centre, toCamera) || offScreen(ndc)
                if (each.pin.hidden) return

                const left = `${((ndc.x * 0.5 + 0.5) * 100).toFixed(3)}%`
                const top = `${((-ndc.y * 0.5 + 0.5) * 100).toFixed(3)}%`
                if (left !== each.left || top !== each.top) {
                    each.left = left
                    each.top = top
                    each.pin.style.left = left
                    each.pin.style.top = top
                }

                const nameplate = callers?.nameplateOf(each.rally.accountId)
                if (nameplate && nameplate.name !== each.shownName) {
                    each.shownName = nameplate.name
                    each.name.textContent = nameplate.name
                    each.name.hidden = nameplate.name === ""
                    for (const [property, value] of Object.entries(nameplate.style)) {
                        each.name.style.setProperty(property, String(value))
                    }
                }
            })

            pointer.update(marked.find((each) => each.own && each.rally.endsAt > now)?.centre, camera)

            return changed
        },

        dispose() {
            unpin()
            geometry.dispose()
            for (const halo of halos) (halo.material as THREE.Material).dispose()
            pointer.dispose()
            group.removeFromParent()
        },
    }
}

function createHalo(geometry: THREE.PlaneGeometry): THREE.Mesh {
    const halo = new THREE.Mesh(geometry, new THREE.ShaderMaterial({
        uniforms: {
            colour: {value: RALLY.clone()},
            opacity: {value: 1},
            edgeAt: {value: 1 / HALO_MARGIN},
        },
        vertexShader: haloVertex,
        fragmentShader: haloFragment,
        transparent: true,
        depthWrite: false,
    }))
    halo.visible = false
    halo.renderOrder = 2
    return halo
}

function layHalo(halo: THREE.Mesh, centre: THREE.Vector3, radius: number) {
    const normal = centre.clone().normalize()
    halo.position.copy(normal).multiplyScalar(1.003)
    halo.quaternion.setFromUnitVectors(Z_AXIS, normal)
    halo.scale.setScalar(radius * HALO_MARGIN)
}

// A flag on a pole, planted where the rally was called: the cloth is the rally's flag, from the tiles' atlas.
function createPin(flag: string, own: boolean): {pin: HTMLElement, name: HTMLElement} {
    const pin = document.createElement("div")
    pin.className = own ? "rally-pin rally-pin--own" : "rally-pin"
    pin.hidden = true

    const pole = document.createElement("span")
    pole.className = "rally-pin-pole"

    const cloth = document.createElement("span")
    cloth.className = "rally-pin-cloth"
    const region = regions.get(flag)
    if (region) Object.assign(cloth.style, clothStyleOf(region))

    const name = document.createElement("span")
    name.className = "rally-pin-name"
    name.hidden = true

    pin.append(pole, cloth, name)
    return {pin, name}
}

// The flag's sprite in the atlas, cut to fill the cloth whole.
export function clothStyleOf(region: Region): Partial<CSSStyleDeclaration> {
    const scale = Math.max(CLOTH.width / region.width, CLOTH.height / region.height)
    const offsetX = (region.width * scale - CLOTH.width) / 2
    const offsetY = (region.height * scale - CLOTH.height) / 2

    return {
        backgroundImage: `url("${ATLAS_URL}")`,
        backgroundSize: `${(ATLAS_SIZE.width * scale).toFixed(2)}px ${(ATLAS_SIZE.height * scale).toFixed(2)}px`,
        backgroundPosition: `${(-region.x * scale - offsetX).toFixed(2)}px ${(-region.y * scale - offsetY).toFixed(2)}px`,
    }
}
