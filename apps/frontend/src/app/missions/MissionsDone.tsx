import {Mission, MissionRates} from "../../backends/missions.ts"
import {xpOf} from "../../domain/missions.ts"
import {t} from "../../i18n/page.ts"
import {usePassing} from "../bonus/BonusAward.tsx"
import {useMissions} from "./missionsStore.ts"
import {partsOf} from "./orderParts.tsx"
import "../bonus/BonusAward.css"
import "./Missions.css"

export default function MissionsDone() {
    const paid = useMissions((missions) => missions.paid)
    const rates = useMissions((missions) => missions.rates)
    const dismissPaid = useMissions((missions) => missions.dismissPaid)

    return paid ? <Card missions={paid} rates={rates} onDone={dismissPaid}/> : null
}

type CardProps = {
    missions: readonly Mission[]
    rates: MissionRates
    onDone: () => void
}

function Card({missions, rates, onDone}: CardProps) {
    usePassing(onDone, missions)

    return <div className="bonus-award" role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <strong className="bonus-award-title">
                {t("mission.missionsDone", {count: missions.length})}
            </strong>
            <ul className="missions-done">
                {missions.map((mission) => {
                    const xp = xpOf(mission.order, rates)
                    return <li key={mission.id} className="missions-done-row">
                        <span className="missions-done-order">{partsOf(mission.order).line}</span>
                        {xp !== undefined && <span className="mission-reward mission-points">{t("mission.xp", {xp})}</span>}
                    </li>
                })}
            </ul>
        </div>
    </div>
}
