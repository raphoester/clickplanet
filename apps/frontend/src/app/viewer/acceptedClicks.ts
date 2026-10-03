import {TileClicker} from "../../backends/backend.ts"

export type ListenForClicks = (onAccepted: () => void) => () => void

export type AcceptedClicks = {
    clicker: TileClicker
    listenForClicks: ListenForClicks
}

export function acceptedClicks(clicker: TileClicker): AcceptedClicks {
    const listeners = new Set<() => void>()
    return {
        clicker: {
            clickTile: async (tileId, countryId, switches) => {
                await clicker.clickTile(tileId, countryId, switches)
                listeners.forEach((listener) => listener())
            },
        },
        listenForClicks: (onAccepted) => {
            listeners.add(onAccepted)
            return () => listeners.delete(onAccepted)
        },
    }
}
