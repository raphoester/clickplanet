import {create} from "zustand"
import {DayMissions, Mission, MissionRates} from "../../backends/missions.ts"
import {newlyPaid} from "../../domain/missions.ts"

export type MissionsState =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", day: DayMissions}

type Missions = {
    state: MissionsState
    // Paid since the day before, until shown.
    paid?: readonly Mission[]
    // What each kind pays in XP; none known shows no XP.
    rates: MissionRates
}

type MissionsStore = Missions & {
    hear: (day: DayMissions) => void
    hearRates: (rates: MissionRates) => void
    fail: () => void
    dismissPaid: () => void
}

export const NO_MISSIONS: Missions = {state: {kind: "loading"}, paid: undefined, rates: {}}

export const useMissions = create<MissionsStore>()((set) => ({
    ...NO_MISSIONS,
    // The first day says nothing paid: what was paid before the page opened was shown then.
    hear: (day) => set(({state, paid}) => {
        const now = state.kind === "ready" ? newlyPaid(state.day.missions, day.missions) : []
        return {state: {kind: "ready", day}, paid: now.length > 0 ? [...(paid ?? []), ...now] : paid}
    }),
    hearRates: (rates) => set({rates}),
    fail: () => set((missions) => missions.state.kind === "ready" ? missions : {state: {kind: "failed"}}),
    dismissPaid: () => set({paid: undefined}),
}))
