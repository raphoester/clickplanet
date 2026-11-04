import {describe, expect, it} from "vitest"
import home from "../index.html?raw"
import privacy from "../privacy.html?raw"
import terms from "../terms.html?raw"
import {DISCORD_JOIN_PATH} from "./domain/discordJoin.ts"

describe("the Discord links", () => {
    for (const [page, html] of Object.entries({home, privacy, terms})) {
        it(`join from the ${page} page by the game's Discord sign-in, never by an invite`, () => {
            expect(html).toContain(`href="${DISCORD_JOIN_PATH}"`)
            expect(html).not.toContain("discord.gg")
            expect(html).not.toContain("discord.com/invite")
        })
    }
})

describe("the social profiles", () => {
    for (const [name, domain] of Object.entries({TIKTOK_PROFILE: "tiktok.com", YOUTUBE_CHANNEL: "youtube.com", INSTAGRAM_PROFILE: "instagram.com"})) {
        it(`${name} reaches the home page from links.ts, never written there by hand`, () => {
            expect(home).toContain(`href="%${name}%"`)
            for (const html of [home, privacy, terms]) expect(html).not.toContain(domain)
        })
    }
})
