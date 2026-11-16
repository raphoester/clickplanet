import {create} from "zustand"
import {DayMissions, Mission} from "../../backends/missions.ts"
import {newlyPaid} from "../../domain/missions.ts"

export type MissionsState =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", day: DayMissions}

type Missions = {
    state: MissionsState
    // Paid since the day before, until shown.
    paid?: readonly Mission[]
}

type MissionsStore = Missions & {
    hear: (day: DayMissions) => void
    fail: () => void
    dismissPaid: () => void
}

export const NO_MISSIONS: Missions = {state: {kind: "loading"}, paid: undefined}

export const useMissions = create<MissionsStore>()((set) => ({
    ...NO_MISSIONS,
    // The first day says nothing paid: what was paid before the page opened was shown then.
    hear: (day) => set(({state, paid}) => {
        const now = state.kind === "ready" ? newlyPaid(state.day.missions, day.missions) : []
        return {state: {kind: "ready", day}, paid: now.length > 0 ? [...(paid ?? []), ...now] : paid}
    }),
    fail: () => set((missions) => missions.state.kind === "ready" ? missions : {state: {kind: "failed"}}),
    dismissPaid: () => set({paid: undefined}),
}))
