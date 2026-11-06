import {create} from "zustand"
import {ALL_OFF, BonusNotice, BonusNoticeEvent, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from "../../domain/bonus.ts"

type BonusStore = {
    award?: BonusReward
    notice?: BonusNoticeEvent
    charges: Charges
    rules?: BonusRules
    switches: Switches
    bombArmed: boolean
    win: (reward: BonusReward) => void
    notify: (notice: BonusNotice) => void
    dismissAward: () => void
    setCharges: (charges: Charges) => void
    setRules: (rules: BonusRules) => void
    setSwitches: (switches: Switches) => void
    setBombArmed: (armed: boolean) => void
}

export const useBonus = create<BonusStore>()((set) => ({
    charges: NO_CHARGES,
    switches: ALL_OFF,
    bombArmed: false,
    win: (award) => set({award}),
    notify: (notice) => set((bonus) => ({notice: {notice, seq: (bonus.notice?.seq ?? 0) + 1}})),
    dismissAward: () => set({award: undefined}),
    setCharges: (charges) => set({charges}),
    setRules: (rules) => set({rules}),
    setSwitches: (switches) => set({switches}),
    setBombArmed: (bombArmed) => set({bombArmed}),
}))
