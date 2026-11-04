import {create} from "zustand"
import {PlayerLine} from "../../backends/player.ts"
import {CountryTiles} from "../../backends/progression.ts"
import {Point} from "../../domain/tilePeek.ts"

export type HolderShown = {
    line: PlayerLine
    playsFor: CountryTiles[]
    mine: boolean
    takenAt: number
    point: Point
    touch: boolean
    id: number
}

type TileHolderStore = {
    holder?: HolderShown
    heldId?: number
    show: (holder: HolderShown | undefined) => void
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
