import {OpenMission} from "../../domain/missionTargets.ts"
import {t} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import {partsOf} from "./orderParts.tsx"
import "./Missions.css"

export default function MissionsHere({missions}: {missions: readonly OpenMission[]}) {
    if (missions.length === 0) return null

    return <ul className="missions-here" aria-label={t("mission.here.label")}>
        {missions.map(({id, order, flag}) => <li key={id} className="missions-here-line">
            <span className="missions-here-ring" aria-hidden="true"/>
            <span>{rich(flag ? "mission.here.flagMission" : "mission.here.mission", {order: partsOf(order).line})}</span>
        </li>)}
    </ul>
}
