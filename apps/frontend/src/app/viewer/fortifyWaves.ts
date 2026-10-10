import * as THREE from "three"
import {FORTIFY_GLOW_SECONDS, FORTIFY_WAVE_SECONDS, MAX_FORTIFY_WAVES, reached, WavePlan, wavePlan} from "../../domain/fortifyWave.ts"

const NEVER = -1e6

export type FortifyUniforms = {
    fortifies: THREE.IUniform<THREE.Vector4[]>
    fortifyLandmass: THREE.IUniform<number[]>
    fortifyReach: THREE.IUniform<number[]>
}

export function fortifyUniforms(): FortifyUniforms {
    return {
        fortifies: {value: Array.from({length: MAX_FORTIFY_WAVES}, () => new THREE.Vector4(0, 0, 1, NEVER))},
        fortifyLandmass: {value: new Array(MAX_FORTIFY_WAVES).fill(-1)},
        fortifyReach: {value: new Array(MAX_FORTIFY_WAVES).fill(0)},
    }
}

export type FortifyWaves = {
    // Shows each tile's shields as the front reaches it, through `reveal`.
    start(landmass: number, tiles: ArrayLike<number>, closing: number, seconds: number): void

    update(seconds: number): boolean

    // Every tile still waiting for the front, shown at once.
    finish(): void
}

type Wave = {
    plan: WavePlan
    startedAt: number
    shown: number
}

export function createFortifyWaves(
    uniforms: FortifyUniforms,
    positions: Float32Array,
    reveal: (tiles: ArrayLike<number>) => void,
): FortifyWaves {
    const slots: (Wave | undefined)[] = new Array(MAX_FORTIFY_WAVES).fill(undefined)
    let next = 0

    const show = (wave: Wave, upTo: number) => {
        if (upTo <= wave.shown) return
        reveal(wave.plan.tiles.subarray(wave.shown, upTo))
        wave.shown = upTo
    }

    return {
        start(landmass, tiles, closing, seconds) {
            const slot = next
            next = (next + 1) % MAX_FORTIFY_WAVES
            const replaced = slots[slot]
            if (replaced) show(replaced, replaced.plan.tiles.length)

            const plan = wavePlan(tiles, positions, closing)
            const at = (closing - 1) * 3
            const centre = new THREE.Vector3(positions[at], positions[at + 1], positions[at + 2]).normalize()
            uniforms.fortifies.value[slot].set(centre.x, centre.y, centre.z, seconds)
            uniforms.fortifyLandmass.value[slot] = landmass
            uniforms.fortifyReach.value[slot] = plan.reach
            slots[slot] = {plan, startedAt: seconds, shown: 0}
        },

        update(seconds) {
            let playing = false
            slots.forEach((wave, slot) => {
                if (!wave) return
                const elapsed = seconds - wave.startedAt
                show(wave, reached(wave.plan, elapsed))
                if (elapsed > FORTIFY_WAVE_SECONDS + FORTIFY_GLOW_SECONDS) {
                    slots[slot] = undefined
                    uniforms.fortifyLandmass.value[slot] = -1
                    uniforms.fortifies.value[slot].w = NEVER
                }
                // The frame the glow ends on is drawn too.
                playing = true
            })
            return playing
        },

        finish() {
            slots.forEach((wave) => {
                if (wave) show(wave, wave.plan.tiles.length)
            })
        },
    }
}
