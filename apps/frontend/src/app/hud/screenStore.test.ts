import {describe, expect, it} from "vitest"
import {MENU_OPEN_KEY, storedOpen} from "./screenStore.ts"

const storage = (entries: Record<string, string>) => () => ({getItem: (key: string) => entries[key] ?? null})

describe("storedOpen", () => {
    it("is unset until the player chose", () => {
        expect(storedOpen(MENU_OPEN_KEY, storage({}))).toBeUndefined()
    })

    it("reads the choice kept", () => {
        expect(storedOpen(MENU_OPEN_KEY, storage({[MENU_OPEN_KEY]: "false"}))).toBe(false)
        expect(storedOpen(MENU_OPEN_KEY, storage({[MENU_OPEN_KEY]: "true"}))).toBe(true)
    })

    it("is unset when the storage refuses", () => {
        expect(storedOpen(MENU_OPEN_KEY, () => {
            throw new Error("denied")
        })).toBeUndefined()
    })
})
