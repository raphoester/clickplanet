import {useId} from "react"
import {CountryTiles, Fronts} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import "./PlayerFronts.css"

const count = new Intl.NumberFormat()

export default function PlayerFronts({playsFor, playsAgainst}: Fronts) {
    if (playsFor.length === 0 && playsAgainst.length === 0) return null

    return <div className="player-fronts">
        <Front heading="Plays for" countries={playsFor}/>
        <Front heading="Plays against" countries={playsAgainst}/>
    </div>
}

function Front({heading, countries}: {heading: string, countries: CountryTiles[]}) {
    const headingId = useId()
    if (countries.length === 0) return null

    const most = Math.max(...countries.map((country) => country.tiles))
    return <section className="panel-box player-front" aria-labelledby={headingId}>
        <h3 id={headingId} className="player-front-heading">{heading}</h3>
        <ol className="player-front-list">
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
    </section>
}
