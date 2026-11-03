import {describe, expect, it, vi} from "vitest"
import {GHOSTS_STORAGE_KEY, saveGhostTakes, savedGhostTakes} from "./ghostStore.ts"

function storeOf(entries: Record<string, string> = {}) {
    const values = new Map(Object.entries(entries))
    return {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => void values.set(key, value),
        removeItem: (key: string) => void values.delete(key),
        values,
    }
}

describe("the ghosts' store", () => {
    it("gives a reload the tiles the ghosts held", () => {
        const store = storeOf()

        saveGhostTakes([{tile: 4, flag: "ro", until: 9_000}], () => store)

        expect(savedGhostTakes(10, () => store)).toEqual([{tile: 4, flag: "ro", until: 9_000}])
    })

    it("clears the key once no ghost holds a tile", () => {
        const store = storeOf({[GHOSTS_STORAGE_KEY]: "[]"})

        saveGhostTakes([], () => store)

        expect(store.values.has(GHOSTS_STORAGE_KEY)).toBe(false)
    })

    it("reads nothing from a broken entry or a storage that refuses", () => {
        vi.spyOn(console, "error").mockImplementation(() => {})
        const refusing = () => {
            throw new DOMException("denied", "SecurityError")
        }

        expect(savedGhostTakes(10, () => storeOf({[GHOSTS_STORAGE_KEY]: "{not json"}))).toEqual([])
        expect(savedGhostTakes(10, refusing)).toEqual([])
        expect(() => saveGhostTakes([{tile: 1, flag: "ro", until: 1}], refusing)).not.toThrow()
    })
})
