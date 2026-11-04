import {create} from "zustand"
import {PlayerLine} from "../../backends/player.ts"

export type TerritoryPlayer = {
    line: PlayerLine
    tiles: number
}

export type Fortifier = {
    line: PlayerLine
    at: number
}

export type TerritoryPlayers = {
    top: TerritoryPlayer[]
    fortifier: Fortifier | undefined
}

type TerritoryPlayersStore = {
    // The territory the players were last read for.
    landmass?: number
    players?: TerritoryPlayers
    show: (landmass: number, players: TerritoryPlayers) => void
    forget: () => void
}

// Nothing about another territory than the one asked.
export const playersOf = (landmass: number | undefined) =>
    (read: TerritoryPlayersStore) => landmass !== undefined && read.landmass === landmass ? read.players : undefined

export const useTerritoryPlayers = create<TerritoryPlayersStore>()((set) => ({
    show: (landmass, players) => set({landmass, players}),
    forget: () => set({landmass: undefined, players: undefined}),
}))
