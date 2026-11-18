import {describe, expect, it} from "vitest"
import {createFortifyWaves, fortifyUniforms} from "./fortifyWaves.ts"
import {FORTIFY_GLOW_SECONDS, FORTIFY_WAVE_SECONDS} from "../../domain/fortifyWave.ts"
import {NO_PROVINCES} from "../../domain/provinces.ts"

const POSITIONS = Float32Array.of(1, 0, 0, 0.99, 0.1, 0, 0.98, 0.2, 0)

describe("a fortify's wave", () => {
    it("runs over the span of provinces it is given, and over none once it has played", () => {
        const uniforms = fortifyUniforms()
        const waves = createFortifyWaves(uniforms, POSITIONS, () => {})
        expect(uniforms.fortifyProvinces.value.map(({x, y}) => [x, y])).toEqual(Array(4).fill([NO_PROVINCES.first, NO_PROVINCES.last]))

        waves.start({first: 12, last: 15}, Uint32Array.of(1, 2, 3), 1, 10)
        expect([uniforms.fortifyProvinces.value[0].x, uniforms.fortifyProvinces.value[0].y]).toEqual([12, 15])

        waves.update(10 + FORTIFY_WAVE_SECONDS + FORTIFY_GLOW_SECONDS + 0.1)
        expect([uniforms.fortifyProvinces.value[0].x, uniforms.fortifyProvinces.value[0].y]).toEqual([NO_PROVINCES.first, NO_PROVINCES.last])
    })
})
