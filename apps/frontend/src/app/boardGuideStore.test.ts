// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./boardGuideStore.ts")).useBoardGuide

describe("the board guide", () => {
    it("is told once per browser", async () => {
        const guide = await loaded()
        expect(guide.getState().guided).toBe(false)

        guide.getState().markGuided()
        vi.resetModules()

        expect((await loaded()).getState().guided).toBe(true)
    })
})
