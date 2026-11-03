import {useEffect} from "react"
import {COUNTRY_STORAGE_KEY} from "../countryPreference.ts"
import {ListenForClicks} from "./acceptedClicks.ts"

export const FRONT_GUIDE_STORAGE_KEY = "clickplanet-front-guide"

// The tiles a first visit takes before its pages open as any other's.
export const GUIDED_TAKES = 50

type Store = Pick<Storage, "getItem" | "setItem">

const browserStore = (): Store => window.localStorage

// Says whether this page is guided. Before the country is stored: a browser with none is on its first visit, entered from then on.
export function enterFrontGuide(storeOf: () => Store = browserStore): boolean {
    try {
        const store = storeOf()
        const takes = store.getItem(FRONT_GUIDE_STORAGE_KEY)
        if (takes !== null) return Number(takes) < GUIDED_TAKES
        if (store.getItem(COUNTRY_STORAGE_KEY) !== null) return false
        store.setItem(FRONT_GUIDE_STORAGE_KEY, "0")
        return true
    } catch {
        return false
    }
}

export function countGuidedTake(storeOf: () => Store = browserStore) {
    try {
        const store = storeOf()
        const takes = Number(store.getItem(FRONT_GUIDE_STORAGE_KEY) ?? GUIDED_TAKES)
        if (takes < GUIDED_TAKES) store.setItem(FRONT_GUIDE_STORAGE_KEY, String(takes + 1))
    } catch (e) {
        console.error("Could not count a guided take", e)
    }
}

export function useGuidedTakes(guided: boolean, listenForClicks: ListenForClicks) {
    useEffect(() => {
        if (!guided) return
        return listenForClicks(({took}) => {
            if (took) countGuidedTake()
        })
    }, [guided, listenForClicks])
}
