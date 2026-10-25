import {useState} from "react"
import {Mission, MissionError} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {done, orderOf, timeLeft} from "../../domain/missions.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {useNow} from "../season/useNow.ts"
import {MissionsState} from "./useMissions.ts"
import "./Missions.css"

export type MissionsPlaceProps = {
    state: MissionsState
    onCollect: (id: string) => Promise<void>
    onSignIn?: () => void
}

export default function MissionsPlace({state, onCollect, onSignIn}: MissionsPlaceProps) {
    const now = useNow()

    if (state.kind === "loading") return <p className="missions-text" role="status">Loading…</p>
    if (state.kind === "failed") return <p className="missions-text" role="alert">Your missions could not be loaded.</p>
    if (state.day.missions.length === 0) return null

    return <div className="missions">
        <p className="missions-left">Ends in {timeLeft(state.day.endsAt, now)}</p>
        <ol className="missions-list">
            {state.day.missions.map((mission) => <MissionRow key={mission.id}
                                                               mission={mission}
                                                               onCollect={onCollect}
                                                               onSignIn={onSignIn}/>)}
        </ol>
    </div>
}

type MissionRowProps = {
    mission: Mission
    onCollect: (id: string) => Promise<void>
    onSignIn?: () => void
}

function MissionRow({mission, onCollect, onSignIn}: MissionRowProps) {
    const [collecting, setCollecting] = useState(false)
    const [full, setFull] = useState(false)
    const country = Countries.get(mission.target)
    const reached = Math.min(mission.progress, mission.goal)

    const collect = () => {
        setCollecting(true)
        onCollect(mission.id)
            .catch((e) => {
                if (e instanceof MissionError && e.failure === "full") setFull(true)
                else if (e instanceof MissionError && e.failure === "guest") onSignIn?.()
                else console.error("Could not collect the mission", e)
            })
            .finally(() => setCollecting(false))
    }

    return <li className={`panel-box mission${mission.collected ? " mission--collected" : ""}`}>
        <div className="mission-head">
            <span className="mission-order">
                {orderOf(mission)} <CountryFlag code={mission.target}/> {country?.name ?? mission.target}
            </span>
            <span className={`mission-reward mission-reward--${mission.reward.kind}`}
                  aria-label={`Reward: ${mission.reward.amount} ${mission.reward.kind}`}>
                <BonusIcon kind={mission.reward.kind}/>
                {mission.reward.amount > 1 && <span>×{mission.reward.amount}</span>}
            </span>
        </div>
        <div className="mission-foot">
            <div className="mission-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={mission.goal}
                 aria-valuenow={reached}
                 aria-label={orderOf(mission)}>
                <div className="mission-bar-filled" style={{width: `${reached / mission.goal * 100}%`}}/>
            </div>
            <span className="mission-count">{reached}/{mission.goal}</span>
            {mission.collected && <span className="chip mission-done">Collected</span>}
            {!mission.collected && done(mission) && (onSignIn
                ? <button type="button" className="button button-mini" onClick={onSignIn}>Sign in to collect</button>
                : <button type="button"
                          className="button button-mini button-gold"
                          disabled={collecting}
                          onClick={collect}>
                    {full ? "Full" : "Collect"}
                </button>)}
        </div>
    </li>
}
