import {afterEach, describe, expect, it} from "vitest"
import {FakeBackend} from "./fakeBackend.ts"
import {FIXTURE_1} from "../testdata/boards.ts"
import type {Allegiance} from "../domain/allegiance.ts"

const backends: FakeBackend[] = []

function fakePlanet(): {backend: FakeBackend, held: () => Allegiance | undefined} {
    const backend = new FakeBackend(1_000, {board: FIXTURE_1, landmasses: () => new Promise(() => {})})
    backends.push(backend)
    let held: Allegiance | undefined
    backend.listenForAllegiance((allegiance) => held = allegiance)
    return {backend, held: () => held}
}

afterEach(() => backends.splice(0).forEach((backend) => backend.close()))

describe("the fake planet's flag held, when its flag closes", () => {
    it("answers it closed, with no wait, and takes another flag at once", async () => {
        const {backend, held} = fakePlanet()
        await backend.pledgeAllegiance("r-k3p9x2ma0b")

        backend.closeFlags(new Set(["r-k3p9x2ma0b"]))
        expect(held()).toEqual({countryId: "r-k3p9x2ma0b", closed: true})

        await backend.pledgeAllegiance("de")
        expect(held()).toEqual({countryId: "de"})
    })

    it("keeps it playable while the rules lock the flags, and says it closed once they do not", () => {
        const {backend, held} = fakePlanet()
        backend.lockFlags(true)
        void backend.pledgeAllegiance("r-k3p9x2ma0b")

        backend.closeFlags(new Set(["r-k3p9x2ma0b"]))
        expect(held()).toEqual({countryId: "r-k3p9x2ma0b", locked: true})

        backend.lockFlags(false)
        expect(held()?.closed).toBe(true)
    })

    it("refuses a pledge to a closed flag, and plays it again once it is back", async () => {
        const {backend, held} = fakePlanet()
        backend.closeFlags(new Set(["r-k3p9x2ma0b"]))
        await expect(backend.pledgeAllegiance("r-k3p9x2ma0b")).rejects.toThrow()

        backend.closeFlags(new Set())
        await backend.pledgeAllegiance("r-k3p9x2ma0b")
        expect(held()).toEqual({countryId: "r-k3p9x2ma0b"})
    })
})
