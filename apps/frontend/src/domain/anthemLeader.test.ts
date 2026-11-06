import {describe, expect, it} from "vitest";
import {AnthemPick, followLeader, HOLD_MS, NO_PICK} from "./anthemLeader.ts";

describe("followLeader", () => {
    it("plays the first leader at once", () => {
        expect(followLeader(NO_PICK, "fr", "points", 0)).toEqual({playing: "fr", order: "points", challenger: undefined})
    })

    it("keeps playing through an empty board", () => {
        const pick = followLeader(NO_PICK, "fr", "points", 0)
        expect(followLeader(pick, undefined, "points", 1)).toBe(pick)
        expect(followLeader(pick, undefined, "tiles", 1)).toBe(pick)
        expect(followLeader(NO_PICK, undefined, "points", 1)).toBe(NO_PICK)
    })

    it("returns the same pick while the leader stays", () => {
        const pick = followLeader(NO_PICK, "fr", "points", 0)
        expect(followLeader(pick, "fr", "points", 5000)).toBe(pick)
    })

    it("switches only once a new leader has held first place long enough", () => {
        let pick = followLeader(NO_PICK, "fr", "points", 0)
        pick = followLeader(pick, "de", "points", 1000)
        expect(pick).toEqual({playing: "fr", order: "points", challenger: {code: "de", since: 1000}})

        expect(followLeader(pick, "de", "points", 1000 + HOLD_MS - 1)).toBe(pick)
        expect(followLeader(pick, "de", "points", 1000 + HOLD_MS))
            .toEqual({playing: "de", order: "points", challenger: undefined})
    })

    it("forgets a challenger that loses the lead back", () => {
        let pick = followLeader(NO_PICK, "fr", "points", 0)
        pick = followLeader(pick, "de", "points", 1000)
        pick = followLeader(pick, "fr", "points", 2000)
        expect(pick).toEqual({playing: "fr", order: "points", challenger: undefined})

        pick = followLeader(pick, "de", "points", 3000)
        expect(followLeader(pick, "de", "points", 1000 + HOLD_MS)).toBe(pick)
    })

    it("restarts the hold when a different country takes the lead", () => {
        const pick: AnthemPick = {playing: "fr", order: "points", challenger: {code: "de", since: 0}}
        expect(followLeader(pick, "es", "points", HOLD_MS))
            .toEqual({playing: "fr", order: "points", challenger: {code: "es", since: HOLD_MS}})
    })

    it("plays the leader of another order at once, and drops the challenger of the last one", () => {
        const pick: AnthemPick = {playing: "pl", order: "points", challenger: {code: "fr", since: 0}}
        expect(followLeader(pick, "pt", "tiles", 1000))
            .toEqual({playing: "pt", order: "tiles", challenger: undefined})
    })

    it("holds a new leader again once the order is followed", () => {
        let pick = followLeader(NO_PICK, "pl", "points", 0)
        pick = followLeader(pick, "pt", "tiles", 1000)
        pick = followLeader(pick, "hu", "tiles", 2000)
        expect(pick).toEqual({playing: "pt", order: "tiles", challenger: {code: "hu", since: 2000}})
    })

    it("keeps the anthem when both orders have the same leader", () => {
        const pick = followLeader(NO_PICK, "pl", "points", 0)
        expect(followLeader(pick, "pl", "tiles", 1000))
            .toEqual({playing: "pl", order: "tiles", challenger: undefined})
    })
})
