import {describe, expect, it} from "vitest"
import {isValidName, parseStoredIdentity, resolveIdentity} from "./chatIdentity.ts"

describe("parseStoredIdentity", () => {
    it("reads back what was stored", () => {
        const raw = JSON.stringify({authorId: "author-1", name: "Ana"})

        expect(parseStoredIdentity(raw)).toEqual({authorId: "author-1", name: "Ana"})
    })

    it("trims the stored name", () => {
        const raw = JSON.stringify({authorId: "author-1", name: "  Ana  "})

        expect(parseStoredIdentity(raw)?.name).toBe("Ana")
    })

    it("keeps the author id and drops a name the server would refuse", () => {
        const raw = JSON.stringify({authorId: "author-1", name: "x".repeat(25)})

        expect(parseStoredIdentity(raw)).toEqual({authorId: "author-1", name: ""})
    })

    it("gives up on anything it cannot read", () => {
        expect(parseStoredIdentity(null)).toBeUndefined()
        expect(parseStoredIdentity("")).toBeUndefined()
        expect(parseStoredIdentity("not json")).toBeUndefined()
        expect(parseStoredIdentity("42")).toBeUndefined()
        expect(parseStoredIdentity(JSON.stringify({name: "Ana"}))).toBeUndefined()
        expect(parseStoredIdentity(JSON.stringify({authorId: ""}))).toBeUndefined()
        expect(parseStoredIdentity(JSON.stringify({authorId: 7}))).toBeUndefined()
    })
})

describe("resolveIdentity", () => {
    it("mints an author id on a first visit, and no name yet", () => {
        const identity = resolveIdentity(null)

        expect(identity.authorId).not.toBe("")
        expect(identity.name).toBe("")
    })

    it("mints a different id per visitor", () => {
        expect(resolveIdentity(null).authorId).not.toBe(resolveIdentity(null).authorId)
    })

    it("keeps the stored id, so a returning player stays the same author", () => {
        const raw = JSON.stringify({authorId: "author-1", name: "Ana"})

        expect(resolveIdentity(raw).authorId).toBe("author-1")
    })
})

describe("isValidName", () => {
    it("wants something that is not blank", () => {
        expect(isValidName("Ana")).toBe(true)
        expect(isValidName("")).toBe(false)
        expect(isValidName("   ")).toBe(false)
    })

    it("bounds the name in runes, as the server does", () => {
        expect(isValidName("x".repeat(24))).toBe(true)
        expect(isValidName("x".repeat(25))).toBe(false)
        expect(isValidName("🌍".repeat(24))).toBe(true)
    })
})
