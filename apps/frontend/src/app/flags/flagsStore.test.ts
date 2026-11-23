import {afterEach, describe, expect, it, vi} from "vitest"
import {flagCatalog, knowsFlag, useFlags} from "./flagsStore.ts"
import type {ListedFlag} from "../../domain/flags.ts"
import {ENGLISH_TRANSLATOR} from "../../i18n/translator.ts"
import {FakeFlagsBackend} from "../../backends/fakeFlagsBackend.ts"

const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}

const settled = () => new Promise((resolve) => setTimeout(resolve, 0))

// A list the test answers by hand, one read at a time.
function listOnHold() {
    const pending: {resolve: (listed: ListedFlag[]) => void, reject: (e: Error) => void}[] = []
    const listFlags = vi.fn<(fresh: boolean) => Promise<ListedFlag[]>>(
        () => new Promise((resolve, reject) => pending.push({resolve, reject})))
    const answer = async (listed: ListedFlag[]) => {
        pending.shift()?.resolve(listed)
        await settled()
    }
    const fail = async () => {
        pending.shift()?.reject(new Error("unreachable"))
        await settled()
    }
    return {listFlags, answer, fail}
}

afterEach(() => useFlags.getState().serve(undefined))

describe("the flags store", () => {
    it("reads the list once at load, which makes an opened flag known by its name", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)

        expect(lister.listFlags.mock.calls).toEqual([[false]])
        expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe(BAVARIA.code)

        await lister.answer([BAVARIA])

        expect(flagCatalog().knows(BAVARIA.code)).toBe(true)
        expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe("Bavaria")
        expect(ENGLISH_TRANSLATOR.countryInText(BAVARIA.code)).toBe("Bavaria")
        expect(lister.listFlags).toHaveBeenCalledTimes(1)
    })

    it("reads once more for a code it does not know, and not again while it stays unknown", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([])

        expect(knowsFlag(BAVARIA.code)).toBe(false)
        expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe(BAVARIA.code)
        expect(lister.listFlags.mock.calls).toEqual([[false], [true]])

        await lister.answer([])
        for (let i = 0; i < 50; i++) {
            expect(knowsFlag(BAVARIA.code)).toBe(false)
            expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe(BAVARIA.code)
        }
        await settled()

        expect(lister.listFlags).toHaveBeenCalledTimes(2)
    })

    it("lets the read in flight answer for a code missed meanwhile", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)

        expect(knowsFlag(BAVARIA.code)).toBe(false)
        await lister.answer([BAVARIA])

        expect(knowsFlag(BAVARIA.code)).toBe(true)
        expect(lister.listFlags).toHaveBeenCalledTimes(1)
    })

    it("reads again past the cache when the list changes, once more behind a read in flight, and no more", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)

        const {changed} = useFlags.getState()
        changed()
        changed()
        changed()
        expect(lister.listFlags).toHaveBeenCalledTimes(1)

        await lister.answer([])
        expect(lister.listFlags.mock.calls).toEqual([[false], [true]])

        await lister.answer([BAVARIA])
        expect(lister.listFlags).toHaveBeenCalledTimes(2)
        expect(flagCatalog().playable(BAVARIA.code)).toBe(true)
    })

    it("keeps the flags it knows when a read fails, and reads on the next change", async () => {
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {})
        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([BAVARIA])

        useFlags.getState().changed()
        await lister.fail()

        expect(flagCatalog().knows(BAVARIA.code)).toBe(true)
        expect(consoleError).toHaveBeenCalledOnce()
        useFlags.getState().changed()
        expect(lister.listFlags).toHaveBeenCalledTimes(3)
        consoleError.mockRestore()
    })

    it("reads nothing with no list served, and forgets the list once it stops", async () => {
        expect(knowsFlag(BAVARIA.code)).toBe(false)

        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([BAVARIA])
        useFlags.getState().serve(undefined)

        expect(flagCatalog().knows(BAVARIA.code)).toBe(false)
        expect(useFlags.getState().read).toBe(false)
    })

    it("follows the fake's flags as planet's stream would tell them", async () => {
        const fake = new FakeFlagsBackend()
        useFlags.getState().serve(fake)
        const stop = fake.listenForFlagChanges(useFlags.getState().changed)
        await vi.waitFor(() => expect(useFlags.getState().read).toBe(true))

        const code = fake.open("Bavaria")
        await vi.waitFor(() => expect(flagCatalog().playable(code)).toBe(true))
        expect(ENGLISH_TRANSLATOR.country(code)).toBe("Bavaria")

        fake.close(code)
        await vi.waitFor(() => expect(flagCatalog().playable(code)).toBe(false))
        expect(flagCatalog().knows(code)).toBe(true)
        stop()
    })
})
