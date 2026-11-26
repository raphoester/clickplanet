import {useEffect} from "react"
import {create} from "zustand"
import {Shop} from "../../backends/shop.ts"
import {Plan, PlanPrice, Renewal, Sale, saleRuns} from "../../domain/shop.ts"

// setTimeout's longest wait: a sale that ends later is checked again then.
const LONGEST_WAIT_MS = 2 ** 31 - 1

type ShopStore = {
    // None while the shop is off, or until a read: then no plan and no pay button shows.
    plans: readonly PlanPrice[]
    // While a sale runs: dropped at its end with no read.
    sale?: Sale
    // The plan the player picked last, in the shop or in the flag's place; none shows the best value.
    chosen?: Plan
    choose: (plan: Plan) => void
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
    let saleEnds: ReturnType<typeof setTimeout> | undefined

    const keepSale = (sale: Sale | undefined) => {
        clearTimeout(saleEnds)
        saleEnds = undefined
        if (!saleRuns(sale, Date.now())) {
            set({sale: undefined})
            return
        }
        set({sale})
        saleEnds = setTimeout(() => keepSale(sale), Math.min(sale.endsAt - Date.now(), LONGEST_WAIT_MS))
    }

    const readPlans = (from: Shop, generation: number) => from.catalog().then(
        ({plans, sale}) => {
            if (generation !== served) return
            set({plans})
            keepSale(sale)
        },
        (e: unknown) => console.error("could not read the plans on sale", e),
    )

    return {
        plans: [],
        choose: (chosen) => set({chosen}),
        renewals: [],
        serve: (next, linked = false) => {
            shop = next
            signedIn = linked
            served += 1
            set({plans: [], renewals: []})
            keepSale(undefined)
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
