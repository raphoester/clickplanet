import * as THREE from "three"
import {Rally} from "../../domain/rallies.ts"
import {createRing, layRing, ringColour, ringOpacity} from "./blasts.ts"
import {createBonusPointer} from "./bonusPointer.ts"

// More rings than this at once and the oldest of the other flags' go undrawn: the page's own come first.
const MAX_RINGS = 24

// --rally, and a pale ring for the other flags'.
const OWN = new THREE.Color(0x0fa3b1)
const OTHER = new THREE.Color(0.86, 0.89, 0.93)

const OWN_OPACITY = 0.95
const OTHER_OPACITY = 0.4

// A ring of a thousand tiles drawn with the bomb's soft edge is a blur: a rally's is a line.
const SHARPNESS = 40

export type RallyRings = {
    readonly object: THREE.Object3D

    // Every ring on, the page's flag's in the rally's colour and the others pale.
    show(rallies: readonly Rally[], flag: string, positions: ArrayLike<number>): void

    // Hides each ring at its end, and points at the newest of the page's flag's when it is off the screen. True when a
    // ring ended, so the frame is drawn again.
    update(now: number, camera: THREE.OrthographicCamera): boolean

    dispose(): void
}

type Shown = {
    rally: Rally
    own: boolean
    centre: THREE.Vector3
}

export function createRallyRings(container: HTMLElement): RallyRings {
    const group = new THREE.Group()
    const geometry = new THREE.PlaneGeometry(2, 2)
    const rings = Array.from({length: MAX_RINGS}, () => {
        const ring = createRing(geometry)
        ;(ring.material as THREE.ShaderMaterial).uniforms.sharpness.value = SHARPNESS
        group.add(ring)
        return ring
    })
    const pointer = createBonusPointer(container, "rally")

    let shown: Shown[] = []

    return {
        object: group,

        show(rallies, flag, positions) {
            shown = [...rallies]
                .sort((a, b) => Number(b.countryId === flag) - Number(a.countryId === flag) || b.endsAt - a.endsAt)
                .slice(0, MAX_RINGS)
                .map((rally) => {
                    const at = (rally.tile - 1) * 3
                    const centre = new THREE.Vector3(positions[at], positions[at + 1], positions[at + 2])
                    return {rally, own: rally.countryId === flag, centre}
                })

            rings.forEach((ring, index) => {
                const each = shown[index]
                ring.visible = each !== undefined
                if (!each) return
                layRing(ring, each.centre, each.rally.radius)
                ringColour(ring, each.own ? OWN : OTHER)
                ringOpacity(ring, each.own ? OWN_OPACITY : OTHER_OPACITY)
            })
        },

        update(now, camera) {
            let changed = false
            shown.forEach((each, index) => {
                const on = each.rally.endsAt > now
                if (rings[index].visible === on) return
                rings[index].visible = on
                changed = true
            })

            pointer.update(shown.find((each) => each.own && each.rally.endsAt > now)?.centre, camera)

            return changed
        },

        dispose() {
            geometry.dispose()
            for (const ring of rings) (ring.material as THREE.Material).dispose()
            pointer.dispose()
            group.removeFromParent()
        },
    }
}
