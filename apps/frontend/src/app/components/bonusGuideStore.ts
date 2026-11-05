import {create} from 'zustand'
import {ChargeKind} from '../../domain/bonus.ts'
import {afterUse, afterWin, BONUS_GUIDE_STORAGE_KEY, BonusGuide, parseBonusGuide} from '../../domain/bonusGuide.ts'

type BonusGuideStore = {
    guide: BonusGuide
    markUsed: (kind: ChargeKind) => void
    markWon: (kind: ChargeKind) => void
}

function readStoredGuide(): string | null {
    try {
        return window.localStorage.getItem(BONUS_GUIDE_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useBonusGuide = create<BonusGuideStore>()((set) => ({
    guide: parseBonusGuide(readStoredGuide()),
    markUsed: (kind) => set(({guide}) => ({guide: afterUse(guide, kind)})),
    markWon: (kind) => set(({guide}) => ({guide: afterWin(guide, kind)})),
}))

useBonusGuide.subscribe(({guide}) => {
    try {
        window.localStorage.setItem(BONUS_GUIDE_STORAGE_KEY, JSON.stringify(guide))
    } catch (e) {
        console.error("Could not persist the bonus guide", e)
    }
})
