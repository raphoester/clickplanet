import {useCallback, useEffect, useState} from 'react'
import {ChargeKind} from '../../domain/bonus.ts'
import {afterUse, afterWin, BONUS_GUIDE_STORAGE_KEY, BonusGuide, parseBonusGuide} from '../../domain/bonusGuide.ts'

function readStoredGuide(): string | null {
    try {
        return window.localStorage.getItem(BONUS_GUIDE_STORAGE_KEY)
    } catch {
        return null
    }
}

export function useBonusGuide() {
    const [guide, setGuide] = useState<BonusGuide>(() => parseBonusGuide(readStoredGuide()))

    useEffect(() => {
        try {
            window.localStorage.setItem(BONUS_GUIDE_STORAGE_KEY, JSON.stringify(guide))
        } catch (e) {
            console.error("Could not persist the bonus guide", e)
        }
    }, [guide])

    const markUsed = useCallback((kind: ChargeKind) => setGuide((was) => afterUse(was, kind)), [])
    const markWon = useCallback((kind: ChargeKind) => setGuide((was) => afterWin(was, kind)), [])

    return {guide, markUsed, markWon}
}
