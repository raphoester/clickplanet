import {describe, expect, it} from "vitest"
import home from "../index.html?raw"
import privacy from "../privacy.html?raw"
import terms from "../terms.html?raw"

describe("the Discord invite", () => {
    for (const [page, html] of Object.entries({home, privacy, terms})) {
        it(`reaches the ${page} page from links.ts, never written there by hand`, () => {
            expect(html).toContain(`href="%DISCORD_INVITE%"`)
            expect(html).not.toContain("discord.gg")
        })
    }
})

describe("the social profiles", () => {
    for (const [name, domain] of Object.entries({TIKTOK_PROFILE: "tiktok.com", INSTAGRAM_PROFILE: "instagram.com"})) {
        it(`${name} reaches the home page from links.ts, never written there by hand`, () => {
            expect(home).toContain(`href="%${name}%"`)
            for (const html of [home, privacy, terms]) expect(html).not.toContain(domain)
        })
    }
})
