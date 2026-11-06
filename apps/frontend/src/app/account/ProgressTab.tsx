import {useId, useState} from "react"
import {Me} from "../../backends/account.ts"
import {PlayerInfoBackend} from "../../backends/player.ts"
import TitleBanner from "../titles/TitleBanner.tsx"
import TitleEmblem from "../titles/TitleEmblem.tsx"
import TrackPath from "../titles/TrackPath.tsx"
import {PlayerStats} from "../players/PlayerCard.tsx"
import {FrontFolds} from "../players/PlayerFronts.tsx"
import Fold from "../components/Fold.tsx"
import {usePlayerInfo} from "../players/usePlayerInfo.ts"
import {useOwnStreak} from "../badges/badgesStore.ts"
import {takenCount} from "../../domain/standings.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useOwnTakes} from "../clicks/useAcceptedClicks.ts"
import {AccountStore} from "./accountStore.ts"
import {useFronts, useTitles} from "./useProgress.ts"

export type ProgressTabProps = {
    store: AccountStore
    me: Me
    stats?: {backend: PlayerInfoBackend, accountId: string}
    listenForClicks?: ListenForClicks
}

// The phone's You sheet. The desktop shows the same in ProfileModal, laid out for its room.
export default function ProgressTab({store, me, stats, listenForClicks}: ProgressTabProps) {
    const [titles, reread] = useTitles(store, me, listenForClicks)
    const fronts = useFronts(store, me, listenForClicks)
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

    const titlesBody = <>
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
    </>

    return <div className="account-progress">
        {worn && <TitleBanner title={worn} compact/>}

        {stats && <OwnStats {...stats} listenForClicks={listenForClicks}/>}

        <Fold title="Titles" startOpen>{titlesBody}</Fold>

        {fronts && <FrontFolds {...fronts}/>}
    </div>
}

type OwnStatsProps = {
    backend: PlayerInfoBackend
    accountId: string
    listenForClicks?: ListenForClicks
}

function OwnStats({backend, accountId, listenForClicks}: OwnStatsProps) {
    const state = usePlayerInfo(backend, {accountId, guest: false})
    const takes = useOwnTakes(listenForClicks)
    useOwnStreak(accountId, state.kind === "ready" ? state.info.streakCurrent : undefined)
    if (state.kind !== "ready") return null
    return <PlayerStats info={{...state.info, tilesTaken: state.info.tilesTaken + takenCount(takes)}}/>
}
