import {describe, expect, it} from "vitest"
import {crazyGamesLanding, invitedCountry, inviteParamsOf, onCrazyGames} from "./crazyGames.ts"
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
