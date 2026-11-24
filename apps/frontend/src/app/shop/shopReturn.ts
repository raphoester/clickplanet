import {useEffect} from "react"
import {awaitsPayment} from "../../domain/shop.ts"
import type {ShopReturn} from "../../domain/shopReturn.ts"
import {useFlagForm} from "../flags/flagFormStore.ts"
import {useFlagSubmissions} from "../flags/submissionsStore.ts"
import {useScreen} from "../hud/screenStore.ts"
import {useNews} from "../news/newsStore.ts"

export const PAID_READ_EVERY_MS = 3_000
export const PAID_READ_FOR_MS = 120_000

// Back from Stripe's page: the news line says so. After a payment, the server opens the flag from Stripe's webhook a
// moment later, so the submissions are read again until none waits for its payment, and the news line tells the
// opening as any other.
export function returnFromShop(back: ShopReturn, now: () => number = Date.now): () => void {
    useFlagForm.getState().close()
    useScreen.getState().setShopOpen(false)
    useNews.getState().tellShop(back)
    if (back !== "paid") return () => {}

    const until = now() + PAID_READ_FOR_MS
    const timer = setInterval(() => {
        const {read, submissions, refresh} = useFlagSubmissions.getState()
        if ((read && !awaitsPayment(submissions)) || now() >= until) {
            clearInterval(timer)
            return
        }
        refresh()
    }, PAID_READ_EVERY_MS)
    return () => clearInterval(timer)
}

export function useShopReturn(back: ShopReturn | undefined): void {
    useEffect(() => back === undefined ? undefined : returnFromShop(back), [back])
}
