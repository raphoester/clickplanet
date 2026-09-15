import {describe, expect, it} from "vitest";
import {AnthemPick, followLeader, HOLD_MS, NO_PICK} from "./anthemLeader.ts";

describe("followLeader", () => {
    it("plays the first leader at once", () => {
        expect(followLeader(NO_PICK, "fr", 0)).toEqual({playing: "fr", challenger: undefined})
    })

    it("keeps playing through an empty board", () => {
        const pick = followLeader(NO_PICK, "fr", 0)
        expect(followLeader(pick, undefined, 1)).toBe(pick)
        expect(followLeader(NO_PICK, undefined, 1)).toBe(NO_PICK)
    })

    it("returns the same pick while the leader stays", () => {
        const pick = followLeader(NO_PICK, "fr", 0)
        expect(followLeader(pick, "fr", 5000)).toBe(pick)
    })

    it("switches only once a new leader has held first place long enough", () => {
        let pick = followLeader(NO_PICK, "fr", 0)
        pick = followLeader(pick, "de", 1000)
        expect(pick).toEqual({playing: "fr", challenger: {code: "de", since: 1000}})

        expect(followLeader(pick, "de", 1000 + HOLD_MS - 1)).toBe(pick)
        expect(followLeader(pick, "de", 1000 + HOLD_MS)).toEqual({playing: "de", challenger: undefined})
    })

    it("forgets a challenger that loses the lead back", () => {
        let pick = followLeader(NO_PICK, "fr", 0)
        pick = followLeader(pick, "de", 1000)
        pick = followLeader(pick, "fr", 2000)
        expect(pick).toEqual({playing: "fr", challenger: undefined})

        // Its next lead starts a new hold rather than counting the old one.
        pick = followLeader(pick, "de", 3000)
        expect(followLeader(pick, "de", 1000 + HOLD_MS)).toBe(pick)
    })

    it("restarts the hold when a different country takes the lead", () => {
        const pick: AnthemPick = {playing: "fr", challenger: {code: "de", since: 0}}
        expect(followLeader(pick, "es", HOLD_MS)).toEqual({playing: "fr", challenger: {code: "es", since: HOLD_MS}})
    })
})
