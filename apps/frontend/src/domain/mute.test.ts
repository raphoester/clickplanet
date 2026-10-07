import {describe, expect, it} from "vitest"
import {describeMute, muteLength} from "./mute.ts"

describe("describeMute", () => {
    it("says one hour for the usual mute", () => {
        expect(describeMute(3600)).toBe("has been muted for one hour")
    })
})

describe("muteLength", () => {
    it.each([
        [3600, "one hour"],
        [7200, "2 hours"],
        [5400, "90 minutes"],
        [60, "one minute"],
        [45, "45 seconds"],
        [86400, "one day"],
        [3 * 86400, "3 days"],
        [7 * 86400, "one week"],
        [90061, "90061 seconds"],
    ])("reads %i seconds as %s", (seconds, words) => {
        expect(muteLength(seconds)).toBe(words)
    })
})
