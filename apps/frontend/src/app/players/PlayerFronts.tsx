import {useId, useState} from "react"
import {CountryTiles} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import CountryFlag from "../components/CountryFlag.tsx"
import "./PlayerFronts.css"

const SHOWN = 3

const count = new Intl.NumberFormat()

export type PlayerFrontsProps = {
    playsFor: CountryTiles[]
    playsAgainst: CountryTiles[]
}

export default function PlayerFronts({playsFor, playsAgainst}: PlayerFrontsProps) {
    if (playsFor.length === 0 && playsAgainst.length === 0) return null

    return <div className="player-fronts">
        <Front heading="Plays for" countries={playsFor}/>
        <Front heading="Plays against" countries={playsAgainst}/>
    </div>
}

function Front({heading, countries}: {heading: string, countries: CountryTiles[]}) {
    const headingId = useId()
    const [open, setOpen] = useState(false)
    if (countries.length === 0) return null

    const total = countries.reduce((sum, country) => sum + country.tiles, 0)
    const shown = open ? countries : countries.slice(0, SHOWN)
    return <section className="panel-box player-front" aria-labelledby={headingId}>
        <h3 id={headingId} className="player-front-heading">{heading}</h3>
        <ol className="player-front-list">
            {shown.map(({countryCode, tiles}) => {
                const name = Countries.get(countryCode)?.name ?? countryCode
                return <li key={countryCode} className="player-front-row">
                    <span className="player-front-line">
                        <CountryFlag code={countryCode}/>
                        <span className="player-front-name" title={name}>{name}</span>
                        <span className="player-front-tiles">{count.format(tiles)}</span>
                    </span>
                    <span className="player-front-bar" aria-hidden="true">
                        <span className="player-front-bar-filled" style={{width: `${tiles / total * 100}%`}}/>
                    </span>
                </li>
            })}
        </ol>
        {countries.length > SHOWN &&
            <button type="button" className="player-front-more" aria-expanded={open} onClick={() => setOpen(!open)}>
                {open ? "Show fewer" : `See all ${countries.length}`}
            </button>}
    </section>
}
