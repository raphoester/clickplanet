// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, renderHook} from "@testing-library/react"
import {PLAIN_TURN} from "../../backends/turn.ts"
import {NO_SCHEDULE, useSchedule} from "../battle/scheduleStore.ts"
import {useGatesFeed} from "./gatesFeed.ts"

afterEach(() => {
    cleanup()
    useSchedule.setState(NO_SCHEDULE)
})

describe("useGatesFeed", () => {
    it("reads the bonuses' rules again when the turn moves to another entry, not on the first turn heard", async () => {
        const grantRules = {refreshGrantRules: vi.fn(async () => {})}
        renderHook(() => useGatesFeed(grantRules))

        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, entryId: "day"}))
        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, entryId: "day", description: "same entry"}))
        expect(grantRules.refreshGrantRules).not.toHaveBeenCalled()

        act(() => useSchedule.getState().hearTurn({...PLAIN_TURN, kind: "battle", entryId: "battle"}))
        expect(grantRules.refreshGrantRules).toHaveBeenCalledOnce()
    })
})
