import {Countries} from "../../domain/countries.ts"
import {freeTilesOf, sharePercent} from "../../domain/territory.ts"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import PointerCard from "../components/PointerCard.tsx"
import RankCoin from "../components/RankCoin.tsx"
import {TerritoryShown} from "./useTerritoryCard.ts"
import "./TerritoryCard.css"

const FLAGS_SHOWN = 5

const count = new Intl.NumberFormat()

export type TerritoryCardProps = {
    shown: TerritoryShown
    flag: string
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function TerritoryCard({shown, flag, onHold, onClose}: TerritoryCardProps) {
    const {territory, point, touch} = shown
    const {name, ground, size, holdings, fortifiedBy, target} = territory
    const free = freeTilesOf(territory)
    const listed = holdings.slice(0, FLAGS_SHOWN)
    const more = holdings.length - listed.length

    return <PointerCard point={point}
                        touch={touch}
                        label={name}
                        className="territory-card"
                        shape={listed.length}
                        onHold={onHold}
                        onClose={onClose}>
        <header className="territory-card-head">
            <h3 className="territory-card-name">
                {ground && <CountryFlag code={ground}/>}
                <span className="territory-card-name-text" title={name}>{name}</span>
            </h3>
            <p className="territory-card-size">
                {tilesOf(size)}{free > 0 && `, ${count.format(free)} free`}
            </p>
        </header>
        {listed.length > 0 && <ol className="territory-card-holdings">
            {listed.map(({flag: holder, tiles}, at) => {
                const country = countryName(holder)
                return <li key={holder} className="territory-card-holding">
                    <span className="territory-card-line">
                        <RankCoin rank={at + 1} you={holder === flag}/>
                        <CountryFlag code={holder}/>
                        <span className="territory-card-country" title={country}>{country}</span>
                        <span className="territory-card-tiles">{count.format(tiles)}</span>
                    </span>
                    <span className="territory-card-share">
                        <span className="territory-card-bar" aria-hidden="true">
                            <span className="territory-card-bar-filled" style={{width: `${tiles / size * 100}%`}}/>
                        </span>
                        <span className="territory-card-percent">{sharePercent(tiles, size)}%</span>
                    </span>
                </li>
            })}
        </ol>}
        {more > 0 && <p className="territory-card-more">+{more} more {more === 1 ? "flag" : "flags"}</p>}
        <div className="pointer-card-foot territory-card-fortify">
            <p className="territory-card-fortify-line">
                <span className="territory-card-shield"><BonusIcon kind="shields"/></span>
                {fortifiedBy
                    ? <span>Fortified by <CountryFlag code={fortifiedBy}/>{countryName(fortifiedBy)}</span>
                    : <span>Never fortified</span>}
            </p>
            {target && <p className="territory-card-fortify-line territory-card-target">
                <Flag code={target.flag}/> {tilesOf(target.missing)} to fortify
            </p>}
        </div>
    </PointerCard>
}

function Flag({code}: {code: string}) {
    const name = countryName(code)
    return <span role="img" aria-label={name} title={name}><CountryFlag code={code}/></span>
}

function countryName(code: string): string {
    return Countries.get(code)?.name ?? code
}

function tilesOf(tiles: number): string {
    return `${count.format(tiles)} ${tiles === 1 ? "tile" : "tiles"}`
}
