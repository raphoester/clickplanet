import {Score} from "../../backends/standings.ts"
import {t, translator} from "../../i18n/page.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import "./Podium.css"

// The first three on gold, silver and bronze steps, 2nd, 1st, 3rd: countries level share a rank and its step.
export default function Podium({places}: {places: Score[]}) {
    const [first, second, third] = places
    const shown = [second, first, third].filter((place): place is Score => place !== undefined)

    return <ol className="podium">
        {shown.map((place) => <li key={place.countryCode} className={`podium-step podium-step-${Math.min(place.rank, 3)}`}>
            <span className="podium-flag"><CountryFlag code={place.countryCode}/></span>
            <span className="podium-country">{translator().country(place.countryCode)}</span>
            <span className="podium-block" aria-label={t("standings.rank", {rank: place.rank})}>
                <span aria-hidden="true">{place.rank}</span>
            </span>
        </li>)}
    </ol>
}
