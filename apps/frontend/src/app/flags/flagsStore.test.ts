import {afterEach, assert, describe, expect, it, vi} from "vitest"
import {flagCatalog, knowsFlag, useFlags} from "./flagsStore.ts"
import type {FlagPicture, ListedFlag} from "../../domain/flags.ts"
import {ENGLISH_TRANSLATOR} from "../../i18n/translator.ts"
import {FakeFlagsBackend, NameRefusal} from "../../backends/fakeFlagsBackend.ts"

const BAVARIA: ListedFlag = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}
const PICTURE: FlagPicture = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}
const SAVOY: ListedFlag = {code: "r-0a1b2c3d4e", name: "Savoy", closed: true, sold: {picture: PICTURE, endsAt: 1, closedAt: 2}}
const DAY_MS = 24 * 60 * 60 * 1000

const settled = () => new Promise((resolve) => setTimeout(resolve, 0))

function codeOf(opened: {code: string} | {refused: NameRefusal}): string {
    assert("code" in opened, `refused: ${JSON.stringify(opened)}`)
    return opened.code
}

// Serves the fake as useFlagsFeed does, and waits for its two reads: the load's and the stream's.
async function serveFake(fake: FakeFlagsBackend) {
    const listFlags = vi.spyOn(fake, "listFlags")
    const {serve, changed, told} = useFlags.getState()
    serve(fake)
    const stop = fake.listenForFlags({onOpened: changed, onChanged: told})
    await vi.waitFor(() => expect(listFlags.mock.calls).toEqual([[false], [true]]))
    return {listFlags, stop}
}

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

    it("reads again past the cache each time the stream opens, once more behind a read in flight, and no more", async () => {
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

    it("takes a flag the stream tells with no read", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([])

        useFlags.getState().told(BAVARIA)
        expect(flagCatalog().playable(BAVARIA.code)).toBe(true)
        expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe("Bavaria")

        useFlags.getState().told({...BAVARIA, closed: true})
        expect(flagCatalog().playable(BAVARIA.code)).toBe(false)
        expect(flagCatalog().knows(BAVARIA.code)).toBe(true)
        expect(lister.listFlags).toHaveBeenCalledTimes(1)
    })

    it("keeps what the stream told during a read over that read's answer, which may predate it", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([])

        useFlags.getState().changed()
        useFlags.getState().told(BAVARIA)
        await lister.answer([])

        expect(flagCatalog().playable(BAVARIA.code)).toBe(true)

        useFlags.getState().changed()
        useFlags.getState().told({...BAVARIA, closed: true})
        await lister.answer([BAVARIA])

        expect(flagCatalog().playable(BAVARIA.code)).toBe(false)
    })

    it("takes a rename the stream tells with no read, wherever the page names the flag", async () => {
        const lister = listOnHold()
        useFlags.getState().serve(lister)
        await lister.answer([BAVARIA, SAVOY])

        useFlags.getState().told({...BAVARIA, name: "Bayern"})
        useFlags.getState().told({...SAVOY, name: "Savoie"})

        expect(flagCatalog().openedName(BAVARIA.code)).toBe("Bayern")
        expect(ENGLISH_TRANSLATOR.country(BAVARIA.code)).toBe("Bayern")
        expect(ENGLISH_TRANSLATOR.countryInText(BAVARIA.code)).toBe("Bayern")
        expect(ENGLISH_TRANSLATOR.country(SAVOY.code)).toBe("Savoie")
        expect(flagCatalog().playableCodes().filter((code) => code === BAVARIA.code)).toHaveLength(1)
        expect(flagCatalog().closedFlag(SAVOY.code)).toBe(true)
        expect(flagCatalog().pictureOf(SAVOY.code)).toEqual(PICTURE)
        expect(lister.listFlags).toHaveBeenCalledTimes(1)
    })

    it("follows the fake's flags as the flags' stream would tell them, reading once more as it opens", async () => {
        let now = 0
        const fake = new FakeFlagsBackend(async () => PICTURE, () => now)
        const {listFlags, stop} = await serveFake(fake)

        const code = codeOf(await fake.open("Bavaria", 7))
        expect(flagCatalog().playable(code)).toBe(true)
        expect(flagCatalog().pictureOf(code)).toEqual(PICTURE)
        expect(flagCatalog().soldOf(code)?.endsAt).toBe(7 * DAY_MS)
        expect(ENGLISH_TRANSLATOR.country(code)).toBe("Bavaria")

        expect(fake.addTime(code, 3)).toBe("longer")
        expect(flagCatalog().soldOf(code)?.endsAt).toBe(10 * DAY_MS)

        fake.end(code)
        expect(flagCatalog().playable(code)).toBe(false)
        expect(flagCatalog().knows(code)).toBe(true)
        expect(flagCatalog().claimable(code)).toBe(true)
        expect(flagCatalog().pictures().get(code)).toEqual(PICTURE)

        now = 400 * DAY_MS
        expect(fake.addTime(code, 5)).toBe("reopened")
        expect(flagCatalog().playable(code)).toBe(true)
        expect(flagCatalog().soldOf(code)?.endsAt).toBe(405 * DAY_MS)

        fake.remove(code, "Broke the rules.")
        expect(flagCatalog().claimable(code)).toBe(false)
        expect(flagCatalog().soldOf(code)?.removed).toBe(true)
        expect(await fake.open("BAVARIA")).toEqual({code: expect.any(String)})
        expect(fake.addTime(code, 5)).toBe("taken")
        expect(listFlags).toHaveBeenCalledTimes(2)
        stop()
    })

    it("renames the fake's flags by the server's rules, as the stream would tell it", async () => {
        const fake = new FakeFlagsBackend(async () => PICTURE, () => 0)
        const {listFlags, stop} = await serveFake(fake)
        const bavaria = codeOf(await fake.open("Bavaria"))
        const savoy = codeOf(await fake.open("Savoy"))

        expect(await fake.open("bavaria")).toEqual({refused: "taken"})
        expect(await fake.open("FRANCE")).toEqual({refused: "taken"})
        expect(fake.rename(bavaria, "savoy")).toBe("taken")
        expect(fake.rename(bavaria, "France")).toBe("taken")
        expect(fake.rename(bavaria, " ")).toBe("empty")
        expect(fake.rename(bavaria, "B".repeat(41))).toBe("too long")
        expect(fake.rename("r-0000000000", "Tyrol")).toBe("unknown")
        expect(ENGLISH_TRANSLATOR.country(bavaria)).toBe("Bavaria")

        expect(fake.rename(bavaria, " Bayern ")).toBe("renamed")
        expect(ENGLISH_TRANSLATOR.country(bavaria)).toBe("Bayern")
        fake.remove(savoy, "Broke the rules.")
        expect(fake.rename(savoy, "savoy")).toBe("renamed")
        expect(ENGLISH_TRANSLATOR.country(savoy)).toBe("savoy")
        expect(listFlags).toHaveBeenCalledTimes(2)
        stop()
    })
})
