import {Fortified} from '../../domain/fortify.ts'
import BonusIcon from './BonusIcon.tsx'
import {useEscape} from './useDialog.ts'
import {t, translator} from '../../i18n/page.ts'
import '../bonus/BonusAward.css'

export type FortifyCardProps = {
    fortified: Fortified
    onDone: () => void
}

export default function FortifyCard({fortified, onDone}: FortifyCardProps) {
    useEscape(onDone)

    const country = translator().country(fortified.fortification.countryId)

    return <div className="bonus-award bonus-award--shields bonus-award--kept" role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <span className="bonus-award-box" aria-hidden="true"><BonusIcon kind="shields"/></span>
            <strong className="bonus-award-title">{t("fortifyCard.title")}</strong>
            <span className="bonus-award-detail">
                {t("fortifyCard.ownsAll", {country, territory: fortified.name})}
            </span>
            <span className="bonus-award-detail">{t("fortifyCard.notTwice")}</span>
            <button type="button" className="button button-gold bonus-award-done" onClick={onDone}>{t("fortifyCard.gotIt")}</button>
        </div>
    </div>
}
