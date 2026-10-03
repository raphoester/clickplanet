import {useEffect, useSyncExternalStore} from "react"
import {AccountState} from "../account/accountStore.ts"
import {SeasonEmailsState, SeasonEmailsStore} from "./seasonEmailsStore.ts"

const HIDDEN: SeasonEmailsState = {kind: "hidden"}
const hidden = () => HIDDEN
const noSubscription = () => () => {}

export function useSeasonEmails(store: SeasonEmailsStore | undefined, account: AccountState): SeasonEmailsState {
    const state = useSyncExternalStore(store?.subscribe ?? noSubscription, store?.state ?? hidden)
    const me = account.kind === "ready" ? account.me : undefined

    useEffect(() => {
        if (store) void store.follow(me)
    }, [store, me])

    useEffect(() => {
        if (!store) return
        const onVisible = () => {
            if (document.visibilityState === "visible") void store.refresh()
        }
        document.addEventListener("visibilitychange", onVisible)
        return () => document.removeEventListener("visibilitychange", onVisible)
    }, [store])

    return state
}
