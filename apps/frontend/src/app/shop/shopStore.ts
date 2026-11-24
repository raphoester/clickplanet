import {useEffect} from "react"
import {create} from "zustand"
import {Shop} from "../../backends/shop.ts"
import type {PlanPrice, Renewal} from "../../domain/shop.ts"

type ShopStore = {
    // None while the shop is off, or until a read: then no plan and no pay button shows.
    plans: readonly PlanPrice[]
    // The account's renewals; none for a guest.
    renewals: readonly Renewal[]
    // None forgets them. A signed-in account's renewals are read too.
    serve: (shop: Shop | undefined, signedIn?: boolean) => void
    refresh: () => void
    // A renewal as an answer says it now.
    renewed: (renewal: Renewal) => void
}

export const useShop = create<ShopStore>()((set) => {
    let shop: Shop | undefined
    let signedIn = false
    let served = 0

    const readPlans = (from: Shop, generation: number) => from.plans().then(
        (plans) => {
            if (generation === served) set({plans})
        },
        (e: unknown) => console.error("could not read the plans on sale", e),
    )

    return {
        plans: [],
        renewals: [],
        serve: (next, linked = false) => {
            shop = next
            signedIn = linked
            served += 1
            set({plans: [], renewals: []})
            if (next !== undefined) void readPlans(next, served)
        },
        refresh: () => {
            const from = shop
            if (from === undefined) return
            const generation = served
            void readPlans(from, generation)
            if (!signedIn) return
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

// The plans for every page, read at once (a GET: none while the shop is off), and a signed-in account's renewals, read
// when the flag's place or the shop opens.
export function useShopFeed(shop: Shop, signedIn: boolean): void {
    useEffect(() => {
        const {serve} = useShop.getState()
        serve(shop, signedIn)
        return () => serve(undefined)
    }, [shop, signedIn])
}
