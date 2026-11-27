import {create} from "zustand"

const DISMISSED_KEY = "clickplanet-flag-rows-dismissed"

// The rows of the account's flags the player put away, by key, kept in this browser. A browser that keeps nothing
// forgets them at the next load.
type DismissedFlagRowsStore = {
    dismissed: ReadonlySet<string>
    dismiss: (key: string) => void
}

function stored(): ReadonlySet<string> {
    try {
        const kept: unknown = JSON.parse(window.localStorage.getItem(DISMISSED_KEY) ?? "[]")
        return new Set(Array.isArray(kept) ? kept.filter((key): key is string => typeof key === "string") : [])
    } catch {
        return new Set()
    }
}

export const useDismissedFlagRows = create<DismissedFlagRowsStore>()((set) => ({
    dismissed: stored(),
    dismiss: (key) => set(({dismissed}) => {
        const next = new Set(dismissed).add(key)
        try {
            window.localStorage.setItem(DISMISSED_KEY, JSON.stringify([...next]))
        } catch {
            // Put away for this visit alone.
        }
        return {dismissed: next}
    }),
}))
