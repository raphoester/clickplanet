import {describe, expect, it, vi} from "vitest"
import {RATIO_STORAGE_KEY, saveRatio, savedRatio} from "./ratioStore.ts"

function storeOf(entries: Record<string, string> = {}) {
    const values = new Map(Object.entries(entries))
    return {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => void values.set(key, value),
        values,
    }
}

const broken = () => {
    throw new DOMException("blocked", "SecurityError")
}

describe("the lowered ratio's store", () => {
    it("gives the next visit the ratio slow frames lowered it to", () => {
        const store = storeOf()

        saveRatio(1.5, () => store)

        expect(store.values.get(RATIO_STORAGE_KEY)).toBe("1.5")
        expect(savedRatio(() => store)).toBe(1.5)
    })

    it("caps nothing when nothing is kept", () => {
        expect(savedRatio(() => storeOf())).toBe(2)
    })

    it("caps nothing for a value it cannot use", () => {
        for (const kept of ["", "soft", "0.5", "3", "NaN"]) {
            expect(savedRatio(() => storeOf({[RATIO_STORAGE_KEY]: kept}))).toBe(2)
        }
    })

    it("caps nothing when the storage cannot be read", () => {
        expect(savedRatio(broken)).toBe(2)
    })

    it("lets a storage it cannot write go", () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {})

        expect(() => saveRatio(1, broken)).not.toThrow()

        error.mockRestore()
    })
})
