import {DayMissions} from "../../backends/missions.ts"
import {missionsOnTile} from "../../domain/missionTargets.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import MissionsHere from "../missions/MissionsHere.tsx"
import PointerCard from "../components/PointerCard.tsx"
import Streak from "../badges/Streak.tsx"
import WornTitle from "../badges/WornTitle.tsx"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {t, translator} from "../../i18n/page.ts"
import {usePlayerCard} from "./playerCardStore.ts"
import {HolderShown} from "./tileHolderStore.ts"
import "./PlayerCard.css"
import "./HolderCard.css"

const NAME_MAX_LENGTH = 18
const FLAGS_SHOWN = 3

export type HolderCardProps = {
    holder: HolderShown
    missions?: DayMissions
    now?: number
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function HolderCard({holder, missions, now = Date.now(), onHold, onClose}: HolderCardProps) {
    const {line, playsFor, mine, takenAt, point, touch} = holder
    const country = translator().country(line.countryCode)
    const name = useWordFilter().masked(line.name)
    const openPlayer = usePlayerCard((card) => card.open)

    return <PointerCard point={point}
                        touch={touch}
                        label={name}
                        className="holder-card"
                        style={authorStyle(line)}
                        onHold={onHold}
                        onClose={onClose}>
        <div className="holder-card-main">
            <WornTitle accountId={line.accountId} size={48}/>
            <div className="holder-card-who">
                <span className="holder-card-name">
                    {!line.guest && line.accountId
                        ? <button type="button"
                                  className="holder-card-name-text player-name-button"
                                  title={name}
                                  onClick={() => {
                                      onClose()
                                      openPlayer(line)
                                  }}>
                            {truncate(name, NAME_MAX_LENGTH)}
                        </button>
                        : <span className="holder-card-name-text" title={name}>{truncate(name, NAME_MAX_LENGTH)}</span>}
                    {line.admin && <AdminCrown/>}
                    <Streak accountId={line.accountId}/>
                </span>
                {playsFor.length > 0 && <span className="holder-card-flags" role="group" aria-label={t("player.foughtFor")}>
                    {playsFor.slice(0, FLAGS_SHOWN).map(({countryCode}) => {
                        const name = translator().country(countryCode)
                        return <span key={countryCode} role="img" aria-label={name} title={name}><CountryFlag code={countryCode}/></span>
                    })}
                </span>}
            </div>
        </div>
        {missions && <MissionsHere missions={missionsOnTile(missions, holder.tile)}/>}
        <p className="pointer-card-foot holder-card-take">
            <CountryFlag code={line.countryCode}/>
            {t(mine ? "holder.youTookIt" : "holder.tookIt", {country, ago: translator().ago(takenAt, now)})}
        </p>
    </PointerCard>
}
