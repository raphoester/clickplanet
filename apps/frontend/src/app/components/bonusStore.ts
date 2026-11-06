import {create} from "zustand"
import {BonusNotice, BonusNoticeEvent, BonusReward} from "../../domain/bonus.ts"

type BonusStore = {
    award?: BonusReward
    notice?: BonusNoticeEvent
    win: (reward: BonusReward) => void
    notify: (notice: BonusNotice) => void
    dismissAward: () => void
}

export const useBonus = create<BonusStore>()((set) => ({
    win: (award) => set({award}),
    notify: (notice) => set((bonus) => ({notice: {notice, seq: (bonus.notice?.seq ?? 0) + 1}})),
    dismissAward: () => set({award: undefined}),
}))
