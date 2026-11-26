import {useCallback, useState} from "react"
import {NudgeScene, SALE_SEEN_STORAGE_KEY, saleKeyOf, saleNudgeDue, SaleSeen, saleSeenOf} from "../../domain/nudges.ts"
import {Sale} from "../../domain/shop.ts"
import {useShop} from "../shop/shopStore.ts"

function readSeen(): SaleSeen | null {
    try {
        return saleSeenOf(window.localStorage.getItem(SALE_SEEN_STORAGE_KEY))
    } catch {
        return null
    }
}

function writeSeen(seen: SaleSeen) {
    try {
        window.localStorage.setItem(SALE_SEEN_STORAGE_KEY, JSON.stringify(seen))
    } catch (e) {
        console.error("Could not remember the sale shown", e)
    }
}

export type SaleNudgeState = {
    // The sale that runs, while its nudge is due.
    due?: Sale
    // The sale ran, whether it was closed or its button pressed.
    seen: (sale: Sale) => void
}

export function useSaleNudge(scene: NudgeScene, now: number): SaleNudgeState {
    const sale = useShop((store) => store.sale)
    const [kept, setKept] = useState(readSeen)

    const seen = useCallback((shown: Sale) => {
        const next = {key: saleKeyOf(shown), at: Date.now()}
        writeSeen(next)
        setKept(next)
    }, [])

    return {due: saleNudgeDue(sale, kept, scene, now) ? sale : undefined, seen}
}
