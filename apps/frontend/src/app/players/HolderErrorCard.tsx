import {DayMissions} from "../../backends/missions.ts"
import {Countries} from "../../domain/countries.ts"
import {missionsOnTile} from "../../domain/missionTargets.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import PointerCard from "../components/PointerCard.tsx"
import MissionsHere from "../missions/MissionsHere.tsx"
import {HolderError} from "./tileHolderStore.ts"
import "./HolderCard.css"

const MESSAGES: Record<HolderError["error"], string> = {
    failed: "Who took this tile could not be loaded.",
    unknown: "No record of who took this tile.",
}

export type HolderErrorCardProps = {
    holder: HolderError
    missions?: DayMissions
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function HolderErrorCard({holder, missions, onHold, onClose}: HolderErrorCardProps) {
    const {error, countryCode, tile, point, touch} = holder
    const country = Countries.get(countryCode)?.name ?? countryCode

    return <PointerCard point={point}
                        touch={touch}
                        label={country}
                        className="holder-card"
                        onHold={onHold}
                        onClose={onClose}>
        <p className="holder-card-main holder-card-error" role="alert">{MESSAGES[error]}</p>
        {missions && <MissionsHere missions={missionsOnTile(missions, tile)}/>}
        <p className="pointer-card-foot holder-card-take">
            <CountryFlag code={countryCode}/>
            {country}
        </p>
    </PointerCard>
}
