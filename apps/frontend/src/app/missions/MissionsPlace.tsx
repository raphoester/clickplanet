import {ReactNode} from "react"
import {Mission, MissionOrder, TileOrder} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {orderOf, timeLeft} from "../../domain/missions.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import {useNow} from "../season/useNow.ts"
import {MissionsState} from "./useMissions.ts"
import "./Missions.css"

export type MissionsPlaceProps = {
    state: MissionsState
    onSignIn?: () => void
}

export default function MissionsPlace({state, onSignIn}: MissionsPlaceProps) {
    const now = useNow()

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

type OrderParts = {
    line: ReactNode
    progress: ReactNode
}

function partsOf(order: MissionOrder): OrderParts {
    switch (order.kind) {
        case "takeFor":
        case "takeFrom":
        case "takeIn":
            return tilePartsOf(order)
    }
}

function tilePartsOf(order: TileOrder): OrderParts {
    const country = Countries.get(order.country)
    const reached = Math.min(order.taken, order.goal)

    return {
        line: <>{orderOf(order)} <CountryFlag code={order.country}/> {country?.name ?? order.country}</>,
        progress: <>
            <div className="mission-bar"
                 role="progressbar"
                 aria-valuemin={0}
                 aria-valuemax={order.goal}
                 aria-valuenow={reached}
                 aria-label={orderOf(order)}>
                <div className="mission-bar-filled" style={{width: `${reached / order.goal * 100}%`}}/>
            </div>
            <span className="mission-count">{reached}/{order.goal}</span>
        </>,
    }
}
