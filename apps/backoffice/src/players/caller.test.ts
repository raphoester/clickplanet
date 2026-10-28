import {describe, expect, it} from "vitest"
import {callerOf, idOf, searchOf, whoOf} from "./caller.ts"

describe("callerOf", () => {
    it("reads a UUID as an account, in lower case", () => {
        expect(callerOf(" 0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10 "))
            .toEqual({accountId: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"})
    })

    it("reads anything else as an address, for the server to check", () => {
        expect(callerOf("203.0.113.7")).toEqual({scope: "203.0.113.7"})
        expect(callerOf("2001:db8:1:2::/64")).toEqual({scope: "2001:db8:1:2::/64"})
    })

    it("reads nothing typed as no player", () => {
        expect(callerOf("  ")).toBeUndefined()
    })
})

describe("naming a caller", () => {
    it("gives its id, and a name only when the page has one", () => {
        expect(idOf({scope: "203.0.113.7"})).toBe("203.0.113.7")
        expect(whoOf({accountId: "a"}, "Ana")).toBe("Ana")
        expect(whoOf({accountId: "a"})).toBe("this account")
        expect(whoOf({scope: "203.0.113.7"})).toBe("this address")
    })
})

describe("searchOf", () => {
    it("opens an account id or an address, and looks for anything else as a name", () => {
        expect(searchOf(" 0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10 "))
            .toEqual({open: {accountId: "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"}})
        expect(searchOf("203.0.113.7")).toEqual({open: {scope: "203.0.113.7"}})
        expect(searchOf("2001:db8:1:2::/64")).toEqual({open: {scope: "2001:db8:1:2::/64"}})
        expect(searchOf(" Brave ")).toEqual({name: "Brave"})
        expect(searchOf("guest_a1b2c3")).toEqual({name: "guest_a1b2c3"})
        expect(searchOf(" ")).toBeUndefined()
    })
})
