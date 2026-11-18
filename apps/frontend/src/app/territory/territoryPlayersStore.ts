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

// A province's last fortify as read: by nobody when it has none.
export type Fortified = {by: Fortifier | undefined}

type TerritoryPlayersStore = {
    // The territory its top players were last read for.
    landmass?: number
    top?: TerritoryPlayer[]
    // The province its last fortify was last read for: a territory card's fortify is a province's.
    province?: number
    fortified?: Fortified
    showTop: (landmass: number, top: TerritoryPlayer[]) => void
    showFortified: (province: number, fortified: Fortified) => void
    forget: () => void
}

// Nothing about another territory than the one asked.
export const topOf = (landmass: number | undefined) =>
    (read: TerritoryPlayersStore) => landmass !== undefined && read.landmass === landmass ? read.top : undefined

// Nothing about another province than the one asked.
export const fortifiedOf = (province: number | undefined) =>
    (read: TerritoryPlayersStore) => province !== undefined && read.province === province ? read.fortified : undefined

export const useTerritoryPlayers = create<TerritoryPlayersStore>()((set) => ({
    showTop: (landmass, top) => set({landmass, top}),
    showFortified: (province, fortified) => set({province, fortified}),
    forget: () => set({landmass: undefined, top: undefined, province: undefined, fortified: undefined}),
}))
