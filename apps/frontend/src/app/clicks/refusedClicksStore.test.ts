import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {useRefusedClicks} from "./refusedClicksStore.ts"

describe("useRefusedClicks", () => {
    beforeEach(() => {
        vi.useFakeTimers()
        useRefusedClicks.setState(useRefusedClicks.getInitialState(), true)
    })
    afterEach(() => vi.useRealTimers())

    it("shakes the meter on a tap out of the flag's reach, which is no dry tap", () => {
        useRefusedClicks.getState().refuseOutOfReach()

        expect(useRefusedClicks.getState()).toMatchObject({refusals: 1, dryTaps: 0})
    })

    it("counts a tap the bank refused as a dry tap", () => {
        useRefusedClicks.getState().refuse()

        expect(useRefusedClicks.getState()).toMatchObject({refusals: 1, dryTaps: 1})
    })
})
