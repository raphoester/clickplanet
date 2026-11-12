import {useId} from "react"
import {CountryTiles, Fronts} from "../../backends/progression.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Fold from "../components/Fold.tsx"
import {Key} from "../../i18n/translator.ts"
import {t, translator} from "../../i18n/page.ts"
import "./PlayerFronts.css"

const FOUGHT_FOR: Key = "player.foughtFor"
const FOUGHT_AGAINST: Key = "player.foughtAgainst"
const FLAGS_IN_SUMMARY = 3

export default function PlayerFronts({playsFor, playsAgainst}: Fronts) {
    if (playsFor.length === 0 && playsAgainst.length === 0) return null

    return <div className="player-fronts">
        <Front heading={t(FOUGHT_FOR)} countries={playsFor}/>
        <Front heading={t(FOUGHT_AGAINST)} countries={playsAgainst}/>
    </div>
}

export function FrontFolds({playsFor, playsAgainst}: Fronts) {
    return <>
        <FrontFold title={t(FOUGHT_FOR)} countries={playsFor}/>
        <FrontFold title={t(FOUGHT_AGAINST)} countries={playsAgainst}/>
    </>
}

function Front({heading, countries}: {heading: string, countries: CountryTiles[]}) {
    const headingId = useId()
    if (countries.length === 0) return null

    return <section className="panel-box player-front" aria-labelledby={headingId}>
        <h3 id={headingId} className="player-front-heading">{heading}</h3>
        <FrontList countries={countries}/>
    </section>
}

function FrontFold({title, countries}: {title: string, countries: CountryTiles[]}) {
    if (countries.length === 0) return null

    const summary = <span className="player-front-summary">
        <span className="player-front-flags">
            {countries.slice(0, FLAGS_IN_SUMMARY).map(({countryCode}) => <CountryFlag key={countryCode} code={countryCode}/>)}
        </span>
        <span className="chip">{translator().number(countries.length)}</span>
    </span>

    return <Fold title={title} summary={summary} summaryLabel={t("player.countries", {count: countries.length})}>
        <div className="panel-box player-front">
            <FrontList countries={countries}/>
        </div>
    </Fold>
}

export function FrontList({countries}: {countries: CountryTiles[]}) {
    const most = Math.max(...countries.map((country) => country.tiles))
    return <ol className="player-front-list">
        {countries.map(({countryCode, tiles}) => {
            const name = translator().country(countryCode)
            return <li key={countryCode} className="player-front-row">
                <span className="player-front-line">
                    <CountryFlag code={countryCode}/>
                    <span className="player-front-name" title={name}>{name}</span>
                    <span className="player-front-tiles">{translator().number(tiles)}</span>
                </span>
                <span className="player-front-bar" aria-hidden="true">
                    <span className="player-front-bar-filled" style={{width: `${tiles / most * 100}%`}}/>
                </span>
            </li>
        })}
    </ol>
}
