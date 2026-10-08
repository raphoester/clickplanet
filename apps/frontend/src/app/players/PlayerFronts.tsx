import {useId} from "react"
import {CountryTiles, Fronts} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import Fold from "../components/Fold.tsx"
import "./PlayerFronts.css"

const FOUGHT_FOR = "Fought for"
const FOUGHT_AGAINST = "Fought against"
const FLAGS_IN_SUMMARY = 3

const count = new Intl.NumberFormat()

export default function PlayerFronts({playsFor, playsAgainst}: Fronts) {
    if (playsFor.length === 0 && playsAgainst.length === 0) return null

    return <div className="player-fronts">
        <Front heading={FOUGHT_FOR} countries={playsFor}/>
        <Front heading={FOUGHT_AGAINST} countries={playsAgainst}/>
    </div>
}

export function FrontFolds({playsFor, playsAgainst}: Fronts) {
    return <>
        <FrontFold title={FOUGHT_FOR} countries={playsFor}/>
        <FrontFold title={FOUGHT_AGAINST} countries={playsAgainst}/>
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
        <span className="chip">{countries.length}</span>
    </span>

    return <Fold title={title} summary={summary} summaryLabel={countries.length === 1 ? "1 country" : `${countries.length} countries`}>
        <div className="panel-box player-front">
            <FrontList countries={countries}/>
        </div>
    </Fold>
}

function FrontList({countries}: {countries: CountryTiles[]}) {
    const most = Math.max(...countries.map((country) => country.tiles))
    return <ol className="player-front-list">
        {countries.map(({countryCode, tiles}) => {
            const name = Countries.get(countryCode)?.name ?? countryCode
            return <li key={countryCode} className="player-front-row">
                <span className="player-front-line">
                    <CountryFlag code={countryCode}/>
                    <span className="player-front-name" title={name}>{name}</span>
                    <span className="player-front-tiles">{count.format(tiles)}</span>
                </span>
                <span className="player-front-bar" aria-hidden="true">
                    <span className="player-front-bar-filled" style={{width: `${tiles / most * 100}%`}}/>
                </span>
            </li>
        })}
    </ol>
}
