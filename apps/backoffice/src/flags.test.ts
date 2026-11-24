import {Code} from "@connectrpc/connect"
import {afterEach, describe, expect, it, vi} from "vitest"
import {FakeAdmin} from "./fakeAdmin.ts"
import {learnFlags, listedFlag, readFlags} from "./flags.ts"
import {Flag} from "./gen/grpc/flags/v1/flags_pb.ts"
import {blankFlag, countryLabel, countryName, flagOf} from "./countries.ts"

const BAVARIA = {code: "r-k3p9x2ma0b", name: "Bavaria", closed: false}

afterEach(() => learnFlags([]))

describe("readFlags", () => {
    it("learns every flag the server lists, the closed ones too", async () => {
        const admin = new FakeAdmin()
        admin.flags.listed = [new Flag(BAVARIA), new Flag({code: "r-0a1b2c3d4e", name: "Savoy", closed: true})]

        await readFlags(admin.flags)

        expect(listedFlag(BAVARIA.code)).toEqual(BAVARIA)
        expect(listedFlag("r-0a1b2c3d4e")?.closed).toBe(true)
        expect(admin.asked.filter(({method}) => method === "listFlags")).toHaveLength(1)
    })

    it("keeps the page's own names when the server has no flags module", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listFlags", Code.Unimplemented)
        const error = vi.spyOn(console, "error").mockImplementation(() => {})

        await readFlags(admin.flags)

        expect(listedFlag(BAVARIA.code)).toBeUndefined()
        expect(countryLabel("dz")).toBe("Algeria")
        expect(error).not.toHaveBeenCalled()
        error.mockRestore()
    })

    it("says so on the console when the read fails, and names nothing new", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listFlags", Code.Unavailable)
        const error = vi.spyOn(console, "error").mockImplementation(() => {})

        await readFlags(admin.flags)

        expect(error).toHaveBeenCalledOnce()
        expect(countryLabel(BAVARIA.code)).toBe(BAVARIA.code.toUpperCase())
        error.mockRestore()
    })
})

describe("a flag opened since the build", () => {
    it("reads as its code, with a blank flag, until the list names it", () => {
        expect(countryName(BAVARIA.code)).toBeUndefined()
        expect(countryLabel(BAVARIA.code)).toBe("R-K3P9X2MA0B")
        expect(flagOf(BAVARIA.code)).toBe("")
        expect(blankFlag(BAVARIA.code)).toBe(true)
    })

    it("is named by the list, and still has no emoji", () => {
        learnFlags([BAVARIA])

        expect(countryName(BAVARIA.code)).toBe("Bavaria")
        expect(countryName(" R-K3P9X2MA0B ")).toBe("Bavaria")
        expect(countryLabel(BAVARIA.code)).toBe("Bavaria")
        expect(flagOf(BAVARIA.code)).toBe("")
        expect(blankFlag(BAVARIA.code)).toBe(true)
    })

    it("leaves the game's own flags as they were", () => {
        learnFlags([{code: "fr", name: "Frankreich", closed: false}, BAVARIA])

        expect(countryName("fr")).toBe("France")
        expect(flagOf("fr")).toBe("\u{1F1EB}\u{1F1F7}")
        expect(blankFlag("fr")).toBe(false)
        expect(blankFlag("")).toBe(false)
    })
})
