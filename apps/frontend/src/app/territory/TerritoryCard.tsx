import {PlayerLine} from "../../backends/player.ts"
import {Countries} from "../../domain/countries.ts"
import {freeTilesOf, sharePercent} from "../../domain/territory.ts"
import {agoOf} from "../../domain/tilePeek.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import PointerCard from "../components/PointerCard.tsx"
import RankCoin from "../components/RankCoin.tsx"
import StreakFlame from "../components/StreakFlame.tsx"
import {truncate} from "../truncate.ts"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {TerritoryPlayers} from "./useTerritoryPlayers.ts"
import "./TerritoryCard.css"

const FLAGS_SHOWN = 5

const NAME_MAX_LENGTH = 18

const count = new Intl.NumberFormat()

export type TerritoryCardProps = {
    shown: TerritoryShown
    flag: string
    players?: TerritoryPlayers
    now?: number
    onOpenPlayer?: (player: PlayerLine) => void
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function TerritoryCard({shown, flag, players, now = Date.now(), onOpenPlayer, onHold, onClose}: TerritoryCardProps) {
    const {territory, point, touch} = shown
    const {name, ground, size, holdings, fortifiedBy, target} = territory
    const free = freeTilesOf(territory)
    const listed = holdings.slice(0, FLAGS_SHOWN)
    const more = holdings.length - listed.length
    const top = players?.top ?? []
    const fortifier = players?.fortifier && players.fortifier.line.countryCode === fortifiedBy ? players.fortifier : undefined

    return <PointerCard point={point}
                        touch={touch}
                        label={name}
                        className="territory-card"
                        shape={`${listed.length} ${top.length} ${fortifier !== undefined}`}
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
        {top.length > 0 && <section className="territory-card-players" aria-labelledby={`territory-players-${shown.id}`}>
            <h4 id={`territory-players-${shown.id}`} className="territory-card-heading">Top players</h4>
            <ol className="territory-card-player-list">
                {top.map(({line, tiles}) => <li key={line.accountId} className="territory-card-player" style={authorStyle(line)}>
                    <CountryFlag code={line.countryCode}/>
                    <PlayerName line={line} onOpenPlayer={onOpenPlayer}/>
                    <span className="territory-card-tiles">{count.format(tiles)}</span>
                </li>)}
            </ol>
        </section>}
        <div className="pointer-card-foot territory-card-fortify">
            <p className="territory-card-fortify-line">
                <span className="territory-card-shield"><BonusIcon kind="shields"/></span>
                {fortifiedBy
                    ? <span>Fortified by <CountryFlag code={fortifiedBy}/>{countryName(fortifiedBy)}</span>
                    : <span>Never fortified</span>}
            </p>
            {fortifier && <p className="territory-card-fortify-line territory-card-fortifier" style={authorStyle(fortifier.line)}>
                <PlayerName line={fortifier.line} onOpenPlayer={onOpenPlayer}/>
                <span className="territory-card-ago">{agoOf(fortifier.at, now)}</span>
            </p>}
            {target && <p className="territory-card-fortify-line territory-card-target">
                <Flag code={target.flag}/> {tilesOf(target.missing)} to fortify
            </p>}
        </div>
    </PointerCard>
}

function PlayerName({line, onOpenPlayer}: {line: PlayerLine, onOpenPlayer?: (player: PlayerLine) => void}) {
    const name = <>
        <span className="territory-card-player-name-text">{truncate(line.name, NAME_MAX_LENGTH)}</span>
        {line.admin && <AdminCrown/>}
        <StreakFlame days={line.streak}/>
    </>
    if (!onOpenPlayer || line.guest || !line.accountId) return <span className="territory-card-player-name" title={line.name}>{name}</span>
    return <button type="button" className="territory-card-player-name" title={line.name} onClick={() => onOpenPlayer(line)}>{name}</button>
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
