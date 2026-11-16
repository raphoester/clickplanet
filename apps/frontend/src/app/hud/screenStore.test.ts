import {afterEach, describe, expect, it} from "vitest"
import {MENU_OPEN_KEY, storedOpen, useScreen} from "./screenStore.ts"

afterEach(() => useScreen.setState(useScreen.getInitialState(), true))

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

describe("the dock's popovers", () => {
    it("opens one over the dock at a time", () => {
        useScreen.getState().togglePopover("clicks")
        useScreen.getState().togglePopover("season")
        useScreen.getState().togglePopover("bonuses")

        expect(useScreen.getState().popovers).toEqual({season: true, battle: false, clicks: false, bonuses: true})

        useScreen.getState().togglePopover("clicks")
        expect(useScreen.getState().popovers).toEqual({season: true, battle: false, clicks: true, bonuses: false})
    })

    it("closes the bonuses under a sheet that opens, and keeps them when one closes", () => {
        useScreen.getState().togglePopover("bonuses")
        useScreen.getState().toggleSheet("chat")
        expect(useScreen.getState().popovers.bonuses).toBe(false)

        useScreen.getState().togglePopover("bonuses")
        useScreen.getState().toggleSheet("chat")
        expect(useScreen.getState().popovers.bonuses).toBe(true)

        useScreen.getState().openSheet("board")
        expect(useScreen.getState().popovers.bonuses).toBe(false)
    })
})
