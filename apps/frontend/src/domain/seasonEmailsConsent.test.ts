import {describe, expect, it} from "vitest"
import {SEASON_EMAILS_CONSENT} from "./seasonEmailsConsent.ts"

describe("SEASON_EMAILS_CONSENT", () => {
    it("pins the button's words to the version the server stores", () => {
        expect(SEASON_EMAILS_CONSENT).toEqual({version: "season-emails-1", words: "Email me before the finale"})
    })
})
