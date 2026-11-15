import {create} from "zustand"

type WarningStore = {
    warned: boolean
    // Another account played from this connection at the same time: a reminder, which bans nothing.
    noticed: boolean
    warn: () => void
    notice: () => void
    dismiss: () => void
    dismissNotice: () => void
}

export const useWarning = create<WarningStore>()((set) => ({
    warned: false,
    noticed: false,
    warn: () => set({warned: true}),
    notice: () => set({noticed: true}),
    dismiss: () => set({warned: false}),
    dismissNotice: () => set({noticed: false}),
}))
