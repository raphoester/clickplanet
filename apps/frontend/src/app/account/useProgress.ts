import {useEffect, useState} from "react"
import {Me} from "../../backends/account.ts"
import {Fronts, TitleDashboard} from "../../backends/player.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useReadsAfterClicks} from "../viewer/useAcceptedClicks.ts"
import {AccountStore} from "./accountStore.ts"

// A signed-in player's titles and fronts, read when the account loads and again once its clicks settle. The
// phone's Progress tab and the desktop's profile both show them.
export type Titles =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", dashboard: TitleDashboard}

export function useTitles(store: AccountStore, me: Me, listenForClicks: ListenForClicks | undefined): [Titles, () => void] {
    const [titles, setTitles] = useState<Titles>({kind: "loading"})
    const [read, setRead] = useState(0)
    const reads = useReadsAfterClicks(listenForClicks)

    useEffect(() => {
        let stale = false
        store.titles().then(
            (dashboard) => {
                if (!stale) setTitles({kind: "ready", dashboard})
            },
            (e) => {
                console.error("Could not read the titles", e)
                if (!stale) setTitles((current) => current.kind === "ready" ? current : {kind: "failed"})
            },
        )
        return () => {
            stale = true
        }
    }, [store, me, read, reads])

    return [titles, () => setRead((n) => n + 1)]
}

export function useFronts(store: AccountStore, me: Me, listenForClicks: ListenForClicks | undefined): Fronts | undefined {
    const [fronts, setFronts] = useState<Fronts>()
    const reads = useReadsAfterClicks(listenForClicks)

    useEffect(() => {
        let stale = false
        store.fronts().then(
            (read) => {
                if (!stale) setFronts(read)
            },
            (e) => console.error("Could not read the countries played for and against", e),
        )
        return () => {
            stale = true
        }
    }, [store, me, reads])

    return fronts
}
