import {useEffect} from "react"
import {Mission} from "../../backends/missions.ts"
import {timeLeft} from "../../domain/missions.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import {useNow} from "../season/useNow.ts"
import {partsOf} from "./orderParts.tsx"
import {MissionsState} from "./useMissions.ts"
import "./Missions.css"

export type MissionsPlaceProps = {
    state: MissionsState
    onSignIn?: () => void
    onSeen?: (missions: readonly Mission[]) => void
}

export default function MissionsPlace({state, onSignIn, onSeen}: MissionsPlaceProps) {
    const now = useNow()

    useEffect(() => {
        if (state.kind === "ready") onSeen?.(state.day.missions)
    }, [state, onSeen])

    if (state.kind === "loading") return <p className="missions-text" role="status">Loading…</p>
    if (state.kind === "failed") return <p className="missions-text" role="alert">Your missions could not be loaded.</p>
    if (state.day.missions.length === 0) return null

    return <div className="missions">
        <p className="missions-left">Ends in {timeLeft(state.day.endsAt, now)}</p>
        <ol className="missions-list">
            {state.day.missions.map((mission) => <MissionRow key={mission.id} mission={mission} onSignIn={onSignIn}/>)}
        </ol>
    </div>
}

type MissionRowProps = {
    mission: Mission
    onSignIn?: () => void
}

function MissionRow({mission, onSignIn}: MissionRowProps) {
    const parts = partsOf(mission.order)

    return <li className={`panel-box mission${mission.paid ? " mission--paid" : ""}`}>
        <div className="mission-head">
            <span className="mission-order">{parts.line}</span>
            <span className={`mission-reward mission-reward--${mission.reward.kind}`}
                  aria-label={`Reward: ${mission.reward.amount} ${mission.reward.kind}`}>
                <BonusIcon kind={mission.reward.kind}/>
                {mission.reward.amount > 1 && <span>×{mission.reward.amount}</span>}
            </span>
        </div>
        <div className="mission-foot">
            {parts.progress}
            {mission.done && (!mission.paid && onSignIn
                ? <button type="button" className="button button-mini" onClick={onSignIn}>Sign in to get it</button>
                : <span className="chip mission-done">Done</span>)}
        </div>
    </li>
}
