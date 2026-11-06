import {useEffect, useSyncExternalStore} from "react"
import {AccountState, AccountStore} from "./accountStore.ts"

export function useAccount(store: AccountStore): AccountState {
    const state = useSyncExternalStore(store.subscribe, store.state)

    useEffect(() => {
        if (store.state().kind === "loading") void store.load()
    }, [store])

    return state
}
