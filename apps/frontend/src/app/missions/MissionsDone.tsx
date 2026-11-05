import {Mission} from "../../backends/missions.ts"
import {BonusRules, describeReward} from "../../domain/bonus.ts"
import {awardOf} from "../../domain/missions.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import {usePassing} from "../components/BonusAward.tsx"
import {partsOf} from "./orderParts.tsx"
import "../components/BonusAward.css"
import "./Missions.css"

export type MissionsDoneProps = {
    missions: readonly Mission[]
    rules: Pick<BonusRules, "blastRadius" | "enclosureMaxTiles">
    onDone: () => void
}

export default function MissionsDone({missions, rules, onDone}: MissionsDoneProps) {
    usePassing(onDone, missions)
    const first = missions[0]

    return <div className={`bonus-award bonus-award--${first.reward.kind}`} role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <span className="bonus-award-box" aria-hidden="true"><BonusIcon kind={first.reward.kind}/></span>
            <strong className="bonus-award-title">
                {missions.length === 1 ? "Mission done" : `${missions.length} missions done`}
            </strong>
            <ul className="missions-done">
                {missions.map((mission) => <li key={mission.id} className="missions-done-row">
                    <span className="missions-done-order">{partsOf(mission.order).line}</span>
                    <span className={`chip missions-done-reward mission-reward--${mission.reward.kind}`}>
                        <BonusIcon kind={mission.reward.kind}/>
                        {describeReward(awardOf(mission.reward, rules)).title}
                    </span>
                </li>)}
            </ul>
        </div>
    </div>
}
