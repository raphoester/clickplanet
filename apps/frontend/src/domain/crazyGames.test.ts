import {describe, expect, it} from "vitest"
import {
    crazyGamesLanding,
    crazyGamesUserOf,
    DELETION_GRACE_MS,
    invitedCountry,
    inviteParamsOf,
    leaving,
    onCrazyGames,
} from "./crazyGames.ts"
import {Countries} from "./countries.ts"

describe("onCrazyGames", () => {
    it("is the CrazyGames host, or a build made for it", () => {
        expect(onCrazyGames("crazygames.clickplanet.lol", undefined)).toBe(true)
        expect(onCrazyGames("localhost", "1")).toBe(true)
    })

    it("is never clickplanet.lol", () => {
        expect(onCrazyGames("clickplanet.lol", undefined)).toBe(false)
        expect(onCrazyGames("www.clickplanet.lol", "")).toBe(false)
        expect(onCrazyGames("clickplanet.lol", "0")).toBe(false)
    })
})

describe("crazyGamesLanding", () => {
    it("is tagged CrazyGames and keeps the host the page came from", () => {
        expect(crazyGamesLanding({tag: "rc", referrer: "www.crazygames.com"})).toEqual({tag: "cg", referrer: "www.crazygames.com"})
    })
})

describe("invitedCountry", () => {
    it("is the flag an invite asks to play for", () => {
        expect(invitedCountry(inviteParamsOf("fr"))).toBe(Countries.get("fr"))
        expect(invitedCountry({f: " DE ", czy_invited_by: "someone"})).toBe(Countries.get("de"))
    })

    it("is none for an invite with no flag, or one that is no country", () => {
        expect(invitedCountry(null)).toBeUndefined()
        expect(invitedCountry({})).toBeUndefined()
        expect(invitedCountry({f: "zz"})).toBeUndefined()
    })
})

function tokenFor(claims: object): string {
    const payload = btoa(JSON.stringify(claims)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")
    return `header.${payload}.signature`
}

describe("crazyGamesUserOf", () => {
    it("is the userId the token is for", () => {
        expect(crazyGamesUserOf(tokenFor({userId: "cg-42", gameId: "g", exp: 1}))).toBe("cg-42")
    })

    it("is none for a token that names no user, or is no token", () => {
        expect(crazyGamesUserOf(tokenFor({gameId: "g"}))).toBeUndefined()
        expect(crazyGamesUserOf(tokenFor({userId: ""}))).toBeUndefined()
        expect(crazyGamesUserOf("not-a-token")).toBeUndefined()
        expect(crazyGamesUserOf("a.%%%.c")).toBeUndefined()
    })
})

describe("leaving", () => {
    const at = Date.UTC(2026, 10, 2)

    it("is the user who asked to delete their account, within the grace period", () => {
        expect(leaving({user: "cg-42", at}, "cg-42", at + DELETION_GRACE_MS - 1)).toBe(true)
    })

    it("is nobody else, nor the same user once the account is gone for good", () => {
        expect(leaving({user: "cg-42", at}, "cg-7", at)).toBe(false)
        expect(leaving({user: "cg-42", at}, undefined, at)).toBe(false)
        expect(leaving(undefined, "cg-42", at)).toBe(false)
        expect(leaving({user: "cg-42", at}, "cg-42", at + DELETION_GRACE_MS)).toBe(false)
    })
})
