import {useEffect} from "react"
import {FlagMission, FlagMissionOrder, Mission, MissionOrder} from "../../backends/missions.ts"
import {timeLeft} from "../../domain/missions.ts"
import {MissionTarget, targetOf} from "../../domain/missionTargets.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import {TargetIcon} from "../components/icons.tsx"
import {useNow} from "../season/useNow.ts"
import {partsOf} from "./orderParts.tsx"
import {useMissions} from "./missionsStore.ts"
import {useSeenMissions} from "./seenMissionsStore.ts"
import "./Missions.css"

export type MissionsPlaceProps = {
    onSignIn?: () => void
    onGo?: (target: MissionTarget) => void
}

export default function MissionsPlace({onSignIn, onGo}: MissionsPlaceProps) {
    const state = useMissions((missions) => missions.state)
    const now = useNow()
    const markSeen = useSeenMissions((seenMissions) => seenMissions.markSeen)

    useEffect(() => {
        if (state.kind === "ready") markSeen(state.day.missions)
    }, [state, markSeen])

    if (state.kind === "loading") return <p className="missions-text" role="status">Loading…</p>
    if (state.kind === "failed") return <p className="missions-text" role="alert">Your missions could not be loaded.</p>
    if (state.day.missions.length === 0 && !state.day.flagMission) return onSignIn
        ? <div className="missions">
            <p className="missions-text">Sign in to get missions.</p>
            <button type="button" className="button missions-sign-in" onClick={onSignIn}>Sign in</button>
        </div>
        : <p className="missions-text">No missions right now.</p>

    return <div className="missions">
        <p className="missions-left">Ends in {timeLeft(state.day.endsAt, now)}</p>
        {state.day.flagMission && <FlagMissionRow mission={state.day.flagMission} onGo={onGo}/>}
        <ol className="missions-list">
            {state.day.missions.map((mission) => <MissionRow key={mission.id} mission={mission} onSignIn={onSignIn} onGo={onGo}/>)}
        </ol>
    </div>
}

function FlagMissionRow({mission, onGo}: {mission: FlagMission, onGo?: (target: MissionTarget) => void}) {
    const parts = partsOf(mission.order)

    return <section className={`panel-box mission${mission.done ? " mission--paid" : ""}`}
                    aria-label="Flag mission">
        <p className="mission-kicker">Flag mission</p>
        <div className="mission-head">
            <span className="mission-order">{parts.line}</span>
            <span className="mission-reward mission-points">+{mission.points} points</span>
        </div>
        <div className="mission-foot">
            {parts.progress}
            {mission.done
                ? <span className="chip mission-done">Done</span>
                : <GoButton order={mission.order} onGo={onGo}/>}
        </div>
    </section>
}

type MissionRowProps = {
    mission: Mission
    onSignIn?: () => void
    onGo?: (target: MissionTarget) => void
}

function MissionRow({mission, onSignIn, onGo}: MissionRowProps) {
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
            {mission.done
                ? (!mission.paid && onSignIn
                    ? <button type="button" className="button button-mini" onClick={onSignIn}>Sign in to get it</button>
                    : <span className="chip mission-done">Done</span>)
                : <GoButton order={mission.order} onGo={onGo}/>}
        </div>
    </li>
}

function GoButton({order, onGo}: {order: MissionOrder | FlagMissionOrder, onGo?: (target: MissionTarget) => void}) {
    const target = targetOf(order)
    if (!target || !onGo) return null

    return <button type="button" className="button button-mini button-secondary mission-go" onClick={() => onGo(target)}>
        <TargetIcon size={16}/>
        Go
    </button>
}
