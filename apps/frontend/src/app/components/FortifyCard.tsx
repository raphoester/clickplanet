import {Countries} from '../../domain/countries.ts'
import {Fortified} from '../../domain/fortify.ts'
import BonusIcon from './BonusIcon.tsx'
import {useEscape} from './useDialog.ts'
import './BonusAward.css'

export type FortifyCardProps = {
    fortified: Fortified
    onDone: () => void
}

export default function FortifyCard({fortified, onDone}: FortifyCardProps) {
    useEscape(onDone)

    const country = Countries.get(fortified.fortification.countryId)?.name ?? fortified.fortification.countryId

    return <div className="bonus-award bonus-award--shields bonus-award--kept" role="status" aria-live="polite">
        <div className="bonus-award-card panel">
            <span className="bonus-award-box" aria-hidden="true"><BonusIcon kind="shields"/></span>
            <strong className="bonus-award-title">Fortified!</strong>
            <span className="bonus-award-detail">
                {country} owns all of {fortified.name}, so every tile there gets +1 shield.
            </span>
            <span className="bonus-award-detail">A flag can't fortify the same territory twice in a row.</span>
            <button type="button" className="button button-gold bonus-award-done" onClick={onDone}>Got it</button>
        </div>
    </div>
}
