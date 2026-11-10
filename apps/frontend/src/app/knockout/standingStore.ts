import {create} from "zustand"
import {sameStanding, Standing} from "../../domain/standing.ts"

type StandingStore = {
    // Undefined while the knock-out does not play.
    standing?: Standing
    // The flag whose out window the player closed, to watch.
    watching?: string
    follow: (standing: Standing | undefined) => void
    watch: (flag: string) => void
    // A click the flag could not make: the out window opens again.
    ask: () => void
}

export const useStanding = create<StandingStore>()((set) => ({
    follow: (standing) => set((now) => sameStanding(now.standing, standing) ? now : {standing}),
    watch: (flag) => set({watching: flag}),
    ask: () => set((now) => now.watching === undefined ? now : {watching: undefined}),
}))
