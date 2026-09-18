import {describe, expect, it} from "vitest"
import {parseStoredIdentity, resolveIdentity} from "./chatIdentity.ts"

describe("parseStoredIdentity", () => {
    it("reads back what was stored", () => {
        const raw = JSON.stringify({authorId: "author-1"})

        expect(parseStoredIdentity(raw)).toEqual({authorId: "author-1"})
    })

    it("keeps the author id and drops the name an older build stored", () => {
        const raw = JSON.stringify({authorId: "author-1", name: "Ana"})

        expect(parseStoredIdentity(raw)).toEqual({authorId: "author-1"})
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
    it("mints an author id on a first visit", () => {
        expect(resolveIdentity(null).authorId).not.toBe("")
    })

    it("mints a different id per visitor", () => {
        expect(resolveIdentity(null).authorId).not.toBe(resolveIdentity(null).authorId)
    })

    it("keeps the stored id, so a returning player stays the same author", () => {
        const raw = JSON.stringify({authorId: "author-1"})

        expect(resolveIdentity(raw).authorId).toBe("author-1")
    })
})
