import {useEffect, useState} from "react"
import {Me} from "../../backends/account.ts"
import {Streak} from "../../backends/player.ts"
import {AccountStore} from "./accountStore.ts"

export function useStreak(store: AccountStore, me: Me): Streak | undefined {
    const [streak, setStreak] = useState<Streak>()

    useEffect(() => {
        let stale = false
        setStreak(undefined)
        store.streak().then(
            (read) => {
                if (!stale) setStreak(read)
            },
            (e) => {
                if (!stale) console.error("Could not read the streak", e)
            },
        )
        return () => {
            stale = true
        }
    }, [store, me])

    return streak
}
