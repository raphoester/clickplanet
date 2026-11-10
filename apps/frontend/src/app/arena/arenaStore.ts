import {create} from "zustand"
import {ArenaFlags, sameArenaFlags} from "../../domain/standing.ts"

type ArenaStore = {
    // Undefined on the world's map.
    flags?: ArenaFlags
    // The flag whose out window the player closed, to watch.
    watching?: string
    follow: (flags: ArenaFlags | undefined) => void
    watch: (flag: string) => void
    // A click the flag could not make: the out window opens again.
    ask: () => void
}

export const useArena = create<ArenaStore>()((set) => ({
    follow: (flags) => set((arena) => sameArenaFlags(arena.flags, flags) ? arena : {flags}),
    watch: (flag) => set({watching: flag}),
    ask: () => set((arena) => arena.watching === undefined ? arena : {watching: undefined}),
}))
