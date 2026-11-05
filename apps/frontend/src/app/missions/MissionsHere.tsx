import {OpenMission} from "../../domain/missionTargets.ts"
import {partsOf} from "./orderParts.tsx"
import "./Missions.css"

export default function MissionsHere({missions}: {missions: readonly OpenMission[]}) {
    if (missions.length === 0) return null

    return <ul className="missions-here" aria-label="Your missions here">
        {missions.map(({id, order, flag}) => <li key={id} className="missions-here-line">
            <span className="missions-here-ring" aria-hidden="true"/>
            <span>{flag ? "Flag mission" : "Mission"}: {partsOf(order).line}</span>
        </li>)}
    </ul>
}
