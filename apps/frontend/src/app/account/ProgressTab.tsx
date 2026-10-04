import {useEffect, useId, useState} from "react"
import {Me} from "../../backends/account.ts"
import {NameColor, PlayerInfoBackend, TitleDashboard} from "../../backends/player.ts"
import TitleBanner from "../titles/TitleBanner.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import TrackPath from "../titles/TrackPath.tsx"
import {PlayerStats} from "../players/PlayerCard.tsx"
import {usePlayerInfo} from "../players/usePlayerInfo.ts"
import {takenCount} from "../../domain/standings.ts"
import {ListenForClicks} from "../viewer/acceptedClicks.ts"
import {useOwnTakes, useReadsAfterClicks} from "../viewer/useAcceptedClicks.ts"
import {AccountStore} from "./accountStore.ts"

type Titles =
    | {kind: "loading"}
    | {kind: "failed"}
    | {kind: "ready", dashboard: TitleDashboard}

function useTitles(store: AccountStore, me: Me, listenForClicks: ListenForClicks | undefined): [Titles, () => void] {
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

export type ProgressTabProps = {
    store: AccountStore
    me: Me
    stats?: {backend: PlayerInfoBackend, name: string}
    listenForClicks?: ListenForClicks
}

export default function ProgressTab({store, me, stats, listenForClicks}: ProgressTabProps) {
    const [titles, reread] = useTitles(store, me, listenForClicks)
    const [wearing, setWearing] = useState<string>()
    const labelId = useId()

    if (titles.kind === "loading") return <p className="account-text" role="status">Loading…</p>
    if (titles.kind === "failed") return <p className="account-failure" role="alert">Your titles could not be loaded.</p>

    const {worn, wearable, tracks} = titles.dashboard
    const wear = (id: string) => {
        setWearing(id)
        store.wearTitle(id)
            .then(reread, (e) => console.error("Could not wear the title", e))
            .finally(() => setWearing(undefined))
    }

    return <div className="account-progress">
        {worn && <TitleBanner title={worn} compact/>}

        {stats && <OwnStats {...stats} listenForClicks={listenForClicks}/>}

        {wearable.length > 0 && <div className="account-wear">
            <span className="menu-label" id={labelId}>Wear a title</span>
            <div className="account-wear-options" role="radiogroup" aria-labelledby={labelId}>
                {wearable.map((title) => {
                    const checked = title.id === worn?.id
                    return <button key={title.id}
                                   type="button"
                                   role="radio"
                                   aria-checked={checked}
                                   className="panel-box account-wear-option"
                                   disabled={wearing !== undefined}
                                   onClick={() => {
                                       if (!checked) wear(title.id)
                                   }}>
                        <TitleEmblem title={title} size={44} ribbon/>
                        <span className="account-wear-name">{title.name}</span>
                    </button>
                })}
            </div>
        </div>}

        {tracks.map((track) => <TrackPath key={track.id} track={track}/>)}
    </div>
}

type OwnStatsProps = {
    backend: PlayerInfoBackend
    name: string
    listenForClicks?: ListenForClicks
}

function OwnStats({backend, name, listenForClicks}: OwnStatsProps) {
    const state = usePlayerInfo(backend, {name, countryCode: "", guest: false, admin: false, color: NameColor.UNSPECIFIED, streak: 0})
    const takes = useOwnTakes(listenForClicks)
    if (state.kind !== "ready") return null
    return <PlayerStats info={{...state.info, tilesTaken: state.info.tilesTaken + takenCount(takes)}}/>
}
