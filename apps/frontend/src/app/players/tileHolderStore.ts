import {create} from "zustand"
import {PlayerLine} from "../../backends/player.ts"
import {CountryTiles} from "../../backends/progression.ts"
import {TileFacts} from "../../domain/missionTargets.ts"
import {Point} from "../../domain/tilePeek.ts"

export type HolderShown = {
    line: PlayerLine
    tile: TileFacts
    playsFor: CountryTiles[]
    mine: boolean
    takenAt: number
    point: Point
    touch: boolean
    id: number
}

// "failed": a read threw. "unknown": no take of the tile's flag, so nobody to name.
export type HolderError = {
    error: "failed" | "unknown"
    countryCode: string
    tile: TileFacts
    point: Point
    touch: boolean
    id: number
}

export type HolderCardShown = HolderShown | HolderError

type TileHolderStore = {
    holder?: HolderCardShown
    heldId?: number
    show: (holder: HolderCardShown | undefined) => void
    hold: (held: boolean) => void
    dismiss: () => void
}

// The pointer is on the card shown.
export const holderHeld = (card: TileHolderStore) => card.holder !== undefined && card.heldId === card.holder.id

export const useTileHolder = create<TileHolderStore>()((set) => ({
    show: (holder) => set({holder}),
    hold: (held) => set(({holder}) => ({heldId: held ? holder?.id : undefined})),
    dismiss: () => set({holder: undefined}),
}))
