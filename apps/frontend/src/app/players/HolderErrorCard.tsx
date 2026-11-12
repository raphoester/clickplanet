import {DayMissions} from "../../backends/missions.ts"
import {missionsOnTile} from "../../domain/missionTargets.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import PointerCard from "../components/PointerCard.tsx"
import MissionsHere from "../missions/MissionsHere.tsx"
import {HolderError} from "./tileHolderStore.ts"
import {Key} from "../../i18n/translator.ts"
import {t, translator} from "../../i18n/page.ts"
import "./HolderCard.css"

const MESSAGES: Record<HolderError["error"], Key> = {
    failed: "holder.readFailed",
    unknown: "holder.unknown",
}

export type HolderErrorCardProps = {
    holder: HolderError
    missions?: DayMissions
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function HolderErrorCard({holder, missions, onHold, onClose}: HolderErrorCardProps) {
    const {error, countryCode, tile, point, touch} = holder
    const country = translator().country(countryCode)

    return <PointerCard point={point}
                        touch={touch}
                        label={country}
                        className="holder-card"
                        onHold={onHold}
                        onClose={onClose}>
        <p className="holder-card-main holder-card-error" role="alert">{t(MESSAGES[error])}</p>
        {missions && <MissionsHere missions={missionsOnTile(missions, tile)}/>}
        <p className="pointer-card-foot holder-card-take">
            <CountryFlag code={countryCode}/>
            {country}
        </p>
    </PointerCard>
}
