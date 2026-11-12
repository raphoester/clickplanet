import {describe, expect, it} from "vitest"
import {muteLengthOf} from "./mute.ts"

describe("muteLengthOf", () => {
    it.each([
        [3600, "hour", 1],
        [7200, "hour", 2],
        [5400, "minute", 90],
        [60, "minute", 1],
        [45, "second", 45],
        [86400, "day", 1],
        [3 * 86400, "day", 3],
        [7 * 86400, "week", 1],
        [90061, "second", 90061],
    ])("reads %i seconds as %s × %i", (seconds, unit, count) => {
        expect(muteLengthOf(seconds)).toEqual({unit, count})
    })
})
