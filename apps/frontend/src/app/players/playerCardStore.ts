import {create} from "zustand"
import {PlayerLine} from "../../backends/player.ts"

type PlayerCardStore = {
    player?: PlayerLine
    open: (player: PlayerLine) => void
    close: () => void
}

export const usePlayerCard = create<PlayerCardStore>()((set) => ({
    open: (player) => set({player}),
    close: () => set({player: undefined}),
}))
