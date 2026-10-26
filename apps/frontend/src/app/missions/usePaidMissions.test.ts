// @vitest-environment jsdom
import {describe, expect, it, vi} from "vitest"
import {renderHook} from "@testing-library/react"
import {Mission} from "../../backends/missions.ts"
import {usePaidMissions} from "./usePaidMissions.ts"
import {MissionsState} from "./useMissions.ts"

const mission = (overrides: Partial<Mission>): Mission => ({
    id: "bank",
    kind: "takeFor",
    target: "fr",
    goal: 60,
    progress: 60,
    paid: false,
    reward: {kind: "shields", amount: 10},
    ...overrides,
})

const ready = (...missions: Mission[]): MissionsState => ({
    kind: "ready",
    day: {countryCode: "fr", enlisted: true, endsAt: 0, missions},
})

describe("usePaidMissions", () => {
    it("tells a mission paid since the last read, and nothing paid before the page opened", () => {
        const onPaid = vi.fn()
        const {rerender} = renderHook(({state}) => usePaidMissions(state, onPaid), {
            initialProps: {state: ready(mission({id: "rival", paid: true}), mission({}))},
        })
        expect(onPaid).not.toHaveBeenCalled()

        rerender({state: ready(mission({id: "rival", paid: true}), mission({paid: true}))})

        expect(onPaid).toHaveBeenCalledTimes(1)
        expect(onPaid.mock.calls[0][0].map((paid: Mission) => paid.id)).toEqual(["bank"])
    })
})
