import * as THREE from "three"
import {Rally} from "../../domain/rallies.ts"
import {BonusRules} from "../../domain/bonus.ts"
import {timeLeftTo} from "../../domain/seasonClock.ts"
import {t, translator} from "../../i18n/page.ts"
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

// The globe's north: a halo's clock starts there.
const NORTH = new THREE.Vector3(0, 1, 0)
const EAST = new THREE.Vector3(1, 0, 0)

// The clock's arc moves in steps this small: a frame is drawn at each, about every half second of a 3-minute rally.
const ARC_STEPS = 360

// A tap on a flag shows its card this long, for a screen with no hover.
const TIP_MS = 4_000

// The cloth of the planted flag, in CSS pixels.
export const CLOTH = {width: 40, height: 27} as const

// Who called a rally, by account: its name as the page may show it, and the style that paints it in its color.
export type RallyCallers = {
    ask(accountId: string): void
    nameplateOf(accountId: string): {name: string, style: Record<string, string | number>} | undefined
}

// The rules in force: what a click inside costs, and how long a rally lasts, which the clock is a share of.
export type RallyRules = () => Pick<BonusRules, "rallyCost" | "rallyLastsMs"> | undefined

export type RallyMarks = {
    readonly object: THREE.Object3D

    // Every rally on: a halo over its ring, its flag planted at its centre with its caller's name, the page's flag's in
    // the rally's colour.
    show(rallies: readonly Rally[], flag: string, positions: ArrayLike<number>): void

    // Each mark goes at its rally's end and each clock runs down; each flag follows the camera, and its card says what
    // the rally does; the arrow points at the newest ring of the page's flag off the screen. True when a halo changed, so
    // the frame is drawn again.
    update(now: number, camera: THREE.OrthographicCamera): boolean

    dispose(): void
}

type Marked = {
    rally: Rally
    own: boolean
    centre: THREE.Vector3
    pin: Pin
    // The clock's step shown, and the whole time it counts down from.
    step: number
    lasts: number
    shownName?: string
    left?: string
    top?: string
}

type Pin = {
    root: HTMLElement
    name: HTMLElement
    tip: HTMLElement
    tipping(): boolean
    detach(): void
}

export function createRallyMarks(container: HTMLElement, callers: RallyCallers | undefined, rules: RallyRules): RallyMarks {
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
        for (const each of marked) each.pin.detach()
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
                    const pin = createPin(rally.countryId, own)
                    container.append(pin.root)
                    callers?.ask(rally.accountId)
                    const lasts = Math.max(rules()?.rallyLastsMs ?? 0, rally.endsAt - performance.now(), 1)
                    return {rally, own, centre, pin, lasts, step: -1}
                })

            halos.forEach((halo, index) => {
                const each = marked[index]
                halo.visible = each !== undefined
                if (!each) return
                layHalo(halo, each.centre, each.rally.radius)
                const uniforms = (halo.material as THREE.ShaderMaterial).uniforms
                uniforms.colour.value.copy(RALLY)
                uniforms.left.value = 1
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

                const step = clockStepOf(each.rally.endsAt - now, each.lasts)
                if (on && step !== each.step) {
                    each.step = step
                    ;(halos[index].material as THREE.ShaderMaterial).uniforms.left.value = step / ARC_STEPS
                    changed = true
                }

                const root = each.pin.root
                const ndc = projected.copy(each.centre).project(camera)
                root.hidden = !on || isBehindGlobe(each.centre, toCamera) || offScreen(ndc)
                if (root.hidden) return

                const left = `${((ndc.x * 0.5 + 0.5) * 100).toFixed(3)}%`
                const top = `${((-ndc.y * 0.5 + 0.5) * 100).toFixed(3)}%`
                if (left !== each.left || top !== each.top) {
                    each.left = left
                    each.top = top
                    root.style.left = left
                    root.style.top = top
                }

                const nameplate = callers?.nameplateOf(each.rally.accountId)
                if (nameplate && nameplate.name !== each.shownName) {
                    each.shownName = nameplate.name
                    each.pin.name.textContent = nameplate.name
                    each.pin.name.hidden = nameplate.name === ""
                    for (const [property, value] of Object.entries(nameplate.style)) {
                        each.pin.name.style.setProperty(property, String(value))
                    }
                }

                if (each.pin.tipping()) tell(each, now, rules()?.rallyCost)
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
            left: {value: 1},
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

// Flat on the globe at the centre, its up toward the north, so every clock starts at its top.
function layHalo(halo: THREE.Mesh, centre: THREE.Vector3, radius: number) {
    const normal = centre.clone().normalize()
    const up = NORTH.clone().addScaledVector(normal, -NORTH.dot(normal))
    if (up.lengthSq() < 1e-6) up.copy(EAST).addScaledVector(normal, -EAST.dot(normal))
    up.normalize()
    const right = new THREE.Vector3().crossVectors(up, normal)
    halo.quaternion.setFromRotationMatrix(new THREE.Matrix4().makeBasis(right, up, normal))
    halo.position.copy(normal).multiplyScalar(1.003)
    halo.scale.setScalar(radius * HALO_MARGIN)
}

// A flag on a pole, planted where the rally was called: the cloth is the rally's flag, from the tiles' atlas. A hover,
// or a tap, opens its card; the globe under it hears neither.
function createPin(flag: string, own: boolean): Pin {
    const root = document.createElement("div")
    root.className = own ? "rally-pin rally-pin--own" : "rally-pin"
    root.hidden = true

    const pole = document.createElement("span")
    pole.className = "rally-pin-pole"

    const cloth = document.createElement("span")
    cloth.className = "rally-pin-cloth"
    const region = regions.get(flag)
    if (region) Object.assign(cloth.style, clothStyleOf(region))

    const name = document.createElement("span")
    name.className = "rally-pin-name"
    name.hidden = true

    const tip = document.createElement("span")
    tip.className = "rally-pin-tip"
    tip.setAttribute("role", "tooltip")
    tip.hidden = true

    root.append(pole, cloth, name, tip)

    let hovered = false
    let tappedUntil = 0
    let timer: ReturnType<typeof setTimeout> | undefined
    const showTip = () => {
        tip.hidden = !(hovered || performance.now() < tappedUntil)
    }

    const swallow = (event: Event) => event.stopPropagation()
    for (const type of ["pointerdown", "pointerup", "pointermove", "wheel"] as const) root.addEventListener(type, swallow)
    root.addEventListener("pointerover", () => {
        hovered = true
        showTip()
    })
    root.addEventListener("pointerout", (event) => {
        if (event.relatedTarget instanceof Node && root.contains(event.relatedTarget)) return
        hovered = false
        showTip()
    })
    root.addEventListener("click", (event) => {
        event.stopPropagation()
        tappedUntil = performance.now() + TIP_MS
        showTip()
        clearTimeout(timer)
        timer = setTimeout(showTip, TIP_MS)
    })

    return {
        root, name, tip,
        tipping: () => !tip.hidden,
        detach: () => {
            clearTimeout(timer)
            root.remove()
        },
    }
}

// The card: who called it, for which flag, what a click inside costs, and how long it has left.
function tell(each: Marked, now: number, cost: number | undefined) {
    const player = each.shownName || t("rally.someone")
    const country = translator().country(each.rally.countryId)
    const price = translator().number(cost ?? 1, {maximumFractionDigits: 2})
    const lines = [
        each.own ? t("rally.tip.own", {player}) : t("rally.tip.other", {player, country}),
        each.own ? t("rally.tip.ownCost", {cost: price}) : t("rally.tip.otherCost", {cost: price}),
        t("rally.tip.left", {time: translator().countdown(timeLeftTo(each.rally.endsAt, now))}),
    ]
    const text = lines.join("\n")
    if (each.pin.tip.textContent !== text) each.pin.tip.textContent = text
}

// The clock's step: how much of its time a rally has left, in ARC_STEPS.
export function clockStepOf(leftMs: number, lastsMs: number): number {
    return Math.max(0, Math.min(ARC_STEPS, Math.ceil(leftMs / lastsMs * ARC_STEPS)))
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
