// @vitest-environment jsdom
import {describe, expect, it, vi} from "vitest"
import {renderHook} from "@testing-library/react"
import {FlagMission} from "../../backends/missions.ts"
import {useFlagMissionDone} from "./useFlagMissionDone.ts"
import {MissionsState} from "./useMissions.ts"

const flag = (done: boolean): FlagMission => ({
    id: "flag-22",
    order: {kind: "breakShields", country: "pw", goal: 450, reached: done ? 450 : 449},
    points: 2,
    done,
})

const ready = (flagMission: FlagMission): MissionsState => ({
    kind: "ready",
    day: {endsAt: 0, missions: [], flagMission},
})

describe("useFlagMissionDone", () => {
    it("tells the flag's mission done since the last read", () => {
        const onDone = vi.fn()
        const {rerender} = renderHook(({state}) => useFlagMissionDone(state, onDone), {
            initialProps: {state: ready(flag(false))},
        })

        rerender({state: ready(flag(true))})

        expect(onDone).toHaveBeenCalledTimes(1)
        expect(onDone.mock.calls[0][0]).toEqual(flag(true))
    })

    it("tells nothing of a mission done before the page opened", () => {
        const onDone = vi.fn()
        const {rerender} = renderHook(({state}) => useFlagMissionDone(state, onDone), {
            initialProps: {state: ready(flag(true))},
        })

        rerender({state: ready(flag(true))})

        expect(onDone).not.toHaveBeenCalled()
    })
})
