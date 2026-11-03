import {GhostTake, ghostTakesOf} from "../../domain/ghosts.ts"

export const GHOSTS_STORAGE_KEY = "clickplanet-ghosts"

type Store = Pick<Storage, "getItem" | "setItem" | "removeItem">

const browserStore = (): Store => window.localStorage

export function savedGhostTakes(size: number, storeOf: () => Store = browserStore): GhostTake[] {
    try {
        const saved = storeOf().getItem(GHOSTS_STORAGE_KEY)
        return saved === null ? [] : ghostTakesOf(JSON.parse(saved), size)
    } catch {
        return []
    }
}

export function saveGhostTakes(takes: readonly GhostTake[], storeOf: () => Store = browserStore) {
    try {
        const store = storeOf()
        if (takes.length === 0) store.removeItem(GHOSTS_STORAGE_KEY)
        else store.setItem(GHOSTS_STORAGE_KEY, JSON.stringify(takes))
    } catch (e) {
        console.error("Could not keep the ghosts' tiles", e)
    }
}
