import {afterEach, describe, expect, it} from "vitest"
import {DayMissions, Mission} from "../../backends/missions.ts"
import {NO_MISSIONS, useMissions} from "./missionsStore.ts"

const mission = (overrides: Partial<Mission>): Mission => ({
    id: "bank",
    order: {kind: "takeFor", country: "fr", goal: 60, taken: 60},
    reward: {kind: "shields", amount: 10},
    done: true,
    paid: false,
    ...overrides,
})

const day = (...missions: Mission[]): DayMissions => ({endsAt: 0, missions})

const missions = () => useMissions.getState()

afterEach(() => useMissions.setState(NO_MISSIONS))

describe("useMissions", () => {
    it("tells a mission paid since the day before, and nothing paid before the page opened", () => {
        missions().hear(day(mission({id: "rival", paid: true}), mission({})))
        expect(missions().paid).toBeUndefined()

        missions().hear(day(mission({id: "rival", paid: true}), mission({paid: true})))

        expect(missions().paid?.map((paid) => paid.id)).toEqual(["bank"])
    })

    it("keeps what was paid shown through a day that pays nothing more, until dismissed", () => {
        missions().hear(day(mission({})))
        missions().hear(day(mission({paid: true})))
        missions().hear(day(mission({paid: true})))

        expect(missions().paid?.map((paid) => paid.id)).toEqual(["bank"])

        missions().dismissPaid()
        expect(missions().paid).toBeUndefined()
    })

    it("keeps the day read over a failed read, and fails only with none", () => {
        missions().fail()
        expect(missions().state).toEqual({kind: "failed"})

        missions().hear(day(mission({})))
        missions().fail()

        expect(missions().state).toEqual({kind: "ready", day: day(mission({}))})
    })
})
