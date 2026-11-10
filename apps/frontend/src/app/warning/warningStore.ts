import {create} from "zustand"

type WarningStore = {
    warned: boolean
    warn: () => void
    dismiss: () => void
}

export const useWarning = create<WarningStore>()((set) => ({
    warned: false,
    warn: () => set({warned: true}),
    dismiss: () => set({warned: false}),
}))
