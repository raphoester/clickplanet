import {describe, expect, it} from "vitest"
import {CLEAR_NOTES, CLEAR_NOTES_KEY, ClearNotes} from "./clearNotes.ts"

function memoryStore(initial: Record<string, string> = {}) {
    const items = new Map(Object.entries(initial))
    return {
        getItem: (key: string) => items.get(key) ?? null,
        setItem: (key: string, value: string) => {
            items.set(key, value)
        },
        items,
    }
}

const throwing = {
    getItem: (): string | null => {
        throw new Error("private window")
    },
    setItem: () => {
        throw new Error("private window")
    },
}

describe("ClearNotes", () => {
    it("says it three times, then leaves it to the dust", () => {
        const notes = new ClearNotes(memoryStore())

        for (let i = 0; i < CLEAR_NOTES; i++) {
            expect(notes.due).toBe(true)
            notes.record()
        }

        expect(notes.due).toBe(false)
    })

    it("remembers across page loads", () => {
        const store = memoryStore()
        const first = new ClearNotes(store)
        first.record()
        first.record()

        expect(store.items.get(CLEAR_NOTES_KEY)).toBe("2")
        const second = new ClearNotes(store)
        expect(second.due).toBe(true)
        second.record()
        expect(new ClearNotes(store).due).toBe(false)
    })

    it("reads a count it cannot parse as none", () => {
        expect(new ClearNotes(memoryStore({[CLEAR_NOTES_KEY]: "not a number"})).due).toBe(true)
    })

    it("still stops after three in a page when the storage throws", () => {
        const notes = new ClearNotes(throwing)

        for (let i = 0; i < CLEAR_NOTES; i++) notes.record()

        expect(notes.due).toBe(false)
    })

    it("counts in memory with no storage at all", () => {
        const notes = new ClearNotes(undefined, 1)
        notes.record()
        expect(notes.due).toBe(false)
    })
})
