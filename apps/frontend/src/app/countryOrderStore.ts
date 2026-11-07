import {create} from "zustand"
import {CountryOrder} from "../domain/race.ts"

export const COUNTRY_ORDER_STORAGE_KEY = "clickplanet-country-order"

const ORDERS: readonly CountryOrder[] = ["points", "tiles", "gain"]

type CountryOrderStore = {
    order: CountryOrder
    setOrder: (order: CountryOrder) => void
}

export function parseCountryOrder(stored: string | null): CountryOrder {
    return ORDERS.find((order) => order === stored) ?? "points"
}

function readStoredOrder(): string | null {
    try {
        return window.localStorage.getItem(COUNTRY_ORDER_STORAGE_KEY)
    } catch {
        return null
    }
}

export const useCountryOrder = create<CountryOrderStore>()((set) => ({
    order: parseCountryOrder(readStoredOrder()),
    setOrder: (order) => set({order}),
}))

useCountryOrder.subscribe(({order}) => {
    try {
        window.localStorage.setItem(COUNTRY_ORDER_STORAGE_KEY, order)
    } catch (e) {
        console.error("Could not persist the country order", e)
    }
})
