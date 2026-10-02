import {useEffect, useSyncExternalStore} from "react"
import {AccountState, AccountStore} from "./accountStore.ts"

const HIDDEN: AccountState = {kind: "hidden"}
const hidden = () => HIDDEN
const noSubscription = () => () => {}

export function useAccount(store?: AccountStore): AccountState {
    const state = useSyncExternalStore(store?.subscribe ?? noSubscription, store?.state ?? hidden)

    useEffect(() => {
        if (store && store.state().kind === "loading") void store.load()
    }, [store])

    return state
}
