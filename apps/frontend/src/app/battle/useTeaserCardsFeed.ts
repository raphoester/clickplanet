import {useEffect} from "react"
import {PromoBackend} from "../../backends/promo.ts"
import {useTeaserCards} from "./teaserCardsStore.ts"

// Reads the battles' cards the page shows from promo, and forgets them once it stops.
export function useTeaserCardsFeed(backend: PromoBackend): void {
    useEffect(() => {
        const {serve} = useTeaserCards.getState()
        serve(backend)
        return () => serve(undefined)
    }, [backend])
}
