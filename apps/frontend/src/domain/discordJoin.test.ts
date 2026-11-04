import {describe, expect, it} from "vitest"
import {DISCORD_JOIN_PATH, discordJoinAsked, withoutDiscordJoin} from "./discordJoin.ts"

const ORIGIN = "https://clickplanet.lol"

describe("the Discord join link", () => {
    it("asks the game to join Discord", () => {
        expect(discordJoinAsked(new URL(DISCORD_JOIN_PATH, ORIGIN))).toBe(true)
    })

    it("asks nothing of a plain page, or of another value", () => {
        expect(discordJoinAsked(new URL("/play", ORIGIN))).toBe(false)
        expect(discordJoinAsked(new URL("/play?discord=open", ORIGIN))).toBe(false)
    })

    it("leaves the address bar, and keeps the rest of the query", () => {
        const stripped = withoutDiscordJoin(new URL(`${DISCORD_JOIN_PATH}&f=de`, ORIGIN))

        expect(stripped.search).toBe("?f=de")
        expect(discordJoinAsked(stripped)).toBe(false)
    })
})
