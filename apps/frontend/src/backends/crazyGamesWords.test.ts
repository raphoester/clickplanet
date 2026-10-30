import {beforeEach, describe, expect, it, vi} from "vitest"
import {crazyGamesWords} from "./crazyGamesWords.ts"

function answering(body: string, init: ResponseInit): typeof fetch {
    return async () => new Response(body, init)
}

beforeEach(() => {
    vi.restoreAllMocks()
    vi.spyOn(console, "error").mockImplementation(() => {})
})

describe("crazyGamesWords", () => {
    it("masks the words of the list", async () => {
        const filter = await crazyGamesWords("/words.txt", answering("ass\nfuck\n", {headers: {"content-type": "text/plain; charset=utf-8"}}))
        expect(filter.masked("fuck you ass")).toBe("f**k you a*s")
    })

    it("masks every word when the list is the home page", async () => {
        const filter = await crazyGamesWords("/words.txt", answering("<!doctype html>", {headers: {"content-type": "text/html"}}))
        expect(filter.masked("hello")).toBe("h***o")
    })

    it("masks every word when the list is missing", async () => {
        const filter = await crazyGamesWords("/words.txt", answering("", {status: 404, headers: {"content-type": "text/plain"}}))
        expect(filter.masked("hello")).toBe("h***o")
    })

    it("masks every word when the fetch fails", async () => {
        const filter = await crazyGamesWords("/words.txt", async () => {
            throw new TypeError("offline")
        })
        expect(filter.masked("hello")).toBe("h***o")
    })
})
