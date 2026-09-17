import {useEffect, useSyncExternalStore} from "react"
import {AccountState, AccountStore} from "./accountStore.ts"

const HIDDEN: AccountState = {kind: "hidden"}
const hidden = () => HIDDEN
const noSubscription = () => () => {}

/** The store's state, loaded on first use. No store — the fake backend — is hidden. */
export function useAccount(store?: AccountStore): AccountState {
    const state = useSyncExternalStore(store?.subscribe ?? noSubscription, store?.state ?? hidden)

    useEffect(() => {
        if (store && store.state().kind === "loading") void store.load()
    }, [store])

    return state
}
