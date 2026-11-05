import {create} from "zustand"
import {PlayerLine} from "../../backends/player.ts"

type PlayerCardStore = {
    openable: boolean
    player?: PlayerLine
    setOpenable: (openable: boolean) => void
    open: (player: PlayerLine) => void
    close: () => void
}

export const usePlayerCard = create<PlayerCardStore>()((set) => ({
    openable: false,
    setOpenable: (openable) => set({openable, player: undefined}),
    open: (player) => set({player}),
    close: () => set({player: undefined}),
}))
