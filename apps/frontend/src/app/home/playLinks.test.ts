// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import home from "../../../index.html?raw"
import {DISCORD_JOIN_PATH} from "../../domain/discordJoin.ts"
import {carryQuery} from "./playLinks.ts"

describe("the Play links on the home page", () => {
    const page = () => new DOMParser().parseFromString(home, "text/html")
    const targets = (root: Document) => [...root.querySelectorAll("a")]
        .map((link) => link.getAttribute("href"))
        .filter((href) => href?.startsWith("/play"))

    it("take a shared flag along to the game, and the Discord links keep their join", () => {
        const root = page()
        carryQuery(root, "?f=de")

        expect(targets(root)).toContain("/play?f=de")
        expect(targets(root)).toContain(`${DISCORD_JOIN_PATH}&f=de`)
        expect(targets(root).every((href) => href === "/play?f=de" || href === `${DISCORD_JOIN_PATH}&f=de`)).toBe(true)
    })

    it("stay as they are when the page was opened without a query", () => {
        const root = page()
        carryQuery(root, "")

        expect(targets(root).every((href) => href === "/play" || href === DISCORD_JOIN_PATH)).toBe(true)
    })
})
