import {useEffect} from "react"
import {create} from "zustand"
import {Shop} from "../../backends/shop.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"

type ShopStore = {
    // None while the shop is off, or until a read: then no plan and no pay button shows.
    plans: readonly PlanPrice[]
    // The account's renewals; none for a guest.
    renewals: readonly Renewal[]
    // None forgets them.
    serve: (shop: Shop | undefined) => void
    refresh: () => void
    // A renewal as an answer says it now.
    renewed: (renewal: Renewal) => void
}

export const useShop = create<ShopStore>()((set) => {
    let shop: Shop | undefined
    let served = 0

    return {
        plans: [],
        renewals: [],
        serve: (next) => {
            shop = next
            served += 1
            set({plans: [], renewals: []})
        },
        refresh: () => {
            const from = shop
            if (from === undefined) return
            const generation = served
            from.plans().then(
                (plans) => {
                    if (generation === served) set({plans})
                },
                (e: unknown) => console.error("could not read the plans on sale", e),
            )
            from.myRenewals().then(
                (renewals) => {
                    if (generation === served) set({renewals})
                },
                (e: unknown) => console.error("could not read your renewals", e),
            )
        },
        renewed: (renewal) => set(({renewals}) => ({
            renewals: renewals.map((kept) => kept.id === renewal.id ? renewal : kept),
        })),
    }
})

// A signed-in account's plans and renewals, read when the flag's place opens.
export function useShopFeed(shop: Shop, signedIn: boolean): void {
    useEffect(() => {
        if (!signedIn) return
        const {serve} = useShop.getState()
        serve(shop)
        return () => serve(undefined)
    }, [shop, signedIn])
}
