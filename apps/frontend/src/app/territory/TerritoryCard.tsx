import {DayMissions} from "../../backends/missions.ts"
import {PlayerLine} from "../../backends/player.ts"
import {missionsOnTerritory} from "../../domain/missionTargets.ts"
import {freeTilesOf, sharePercent} from "../../domain/territory.ts"
import {authorStyle} from "../chat/authorStyle.ts"
import AdminCrown from "../components/AdminCrown.tsx"
import BonusIcon from "../components/BonusIcon.tsx"
import CountryFlag from "../components/CountryFlag.tsx"
import PointerCard from "../components/PointerCard.tsx"
import MissionsHere from "../missions/MissionsHere.tsx"
import RankCoin from "../components/RankCoin.tsx"
import StreakFlame from "../components/StreakFlame.tsx"
import {useBadge} from "../badges/badgesStore.ts"
import {truncate} from "../truncate.ts"
import {useWordFilter} from "../crazygames/wordFilter.ts"
import {usePlayerCard} from "../players/playerCardStore.ts"
import {TerritoryShown} from "./useTerritoryCard.ts"
import {TerritoryPlayers} from "./territoryPlayersStore.ts"
import {t, translator} from "../../i18n/page.ts"
import {rich} from "../../i18n/rich.tsx"
import "./TerritoryCard.css"

const FLAGS_SHOWN = 5

const NAME_MAX_LENGTH = 18

export type TerritoryCardProps = {
    shown: TerritoryShown
    flag: string
    players?: TerritoryPlayers
    missions?: DayMissions
    now?: number
    onHold: (held: boolean) => void
    onClose: () => void
}

export default function TerritoryCard({shown, flag, players, missions, now = Date.now(), onHold, onClose}: TerritoryCardProps) {
    const {territory, point, touch} = shown
    const {name, ground, size, holdings, fortifiedBy, target} = territory
    const free = freeTilesOf(territory)
    const listed = holdings.slice(0, FLAGS_SHOWN)
    const more = holdings.length - listed.length
    const top = players?.top ?? []
    const fortifier = players?.fortifier && players.fortifier.line.countryCode === fortifiedBy ? players.fortifier : undefined
    const openPlayer = usePlayerCard((card) => card.open)
    const onOpen = (player: PlayerLine) => {
        onClose()
        openPlayer(player)
    }

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
                {free > 0 ? t("territory.sizeFree", {tiles: size, free}) : t("common.tiles", {count: size})}
            </p>
        </header>
        {listed.length > 0 && <ol className="territory-card-holdings">
            {listed.map(({flag: holder, tiles}, at) => {
                const country = translator().country(holder)
                return <li key={holder} className="territory-card-holding">
                    <span className="territory-card-line">
                        <RankCoin rank={at + 1} you={holder === flag}/>
                        <CountryFlag code={holder}/>
                        <span className="territory-card-country" title={country}>{country}</span>
                        <span className="territory-card-tiles">{translator().number(tiles)}</span>
                    </span>
                    <span className="territory-card-share">
                        <span className="territory-card-bar" aria-hidden="true">
                            <span className="territory-card-bar-filled" style={{width: `${tiles / size * 100}%`}}/>
                        </span>
                        <span className="territory-card-percent">{t("board.mapShare", {share: sharePercent(tiles, size)})}</span>
                    </span>
                </li>
            })}
        </ol>}
        {more > 0 && <p className="territory-card-more">{t("territory.moreFlags", {count: more})}</p>}
        {top.length > 0 && <section className="territory-card-players" aria-labelledby={`territory-players-${shown.id}`}>
            <h4 id={`territory-players-${shown.id}`} className="territory-card-heading">{t("territory.topPlayers")}</h4>
            <ol className="territory-card-player-list">
                {top.map(({line, tiles}) => <li key={`${line.accountId}:${line.countryCode}`} className="territory-card-player" style={authorStyle(line)}>
                    <CountryFlag code={line.countryCode}/>
                    <PlayerName line={line} onOpen={onOpen}/>
                    <span className="territory-card-tiles">{translator().number(tiles)}</span>
                </li>)}
            </ol>
        </section>}
        {missions && <MissionsHere missions={missionsOnTerritory(missions, territory)}/>}
        <div className="pointer-card-foot territory-card-fortify">
            <p className="territory-card-fortify-line">
                <span className="territory-card-shield"><BonusIcon kind="shields"/></span>
                {fortifiedBy
                    ? <span>{rich("territory.fortifiedBy", {flag: <CountryFlag code={fortifiedBy}/>, country: translator().country(fortifiedBy)})}</span>
                    : <span>{t("territory.neverFortified")}</span>}
            </p>
            {fortifier && <p className="territory-card-fortify-line territory-card-fortifier" style={authorStyle(fortifier.line)}>
                <PlayerName line={fortifier.line} onOpen={onOpen}/>
                <span className="territory-card-ago">{translator().ago(fortifier.at, now)}</span>
            </p>}
            {target && <p className="territory-card-fortify-line territory-card-target">
                {rich("territory.toFortify", {flag: <Flag code={target.flag}/>, missing: target.missing})}
            </p>}
        </div>
    </PointerCard>
}

function PlayerName({line, onOpen}: {line: PlayerLine, onOpen: (player: PlayerLine) => void}) {
    const shown = useWordFilter().masked(line.name)
    const text = truncate(shown, NAME_MAX_LENGTH)
    const streak = useBadge(line.accountId)?.streak ?? 0
    // The flame is a button of its own: it stands beside the name's button, never inside it.
    return <span className="territory-card-player-name">
        {line.guest || !line.accountId
            ? <span className="territory-card-player-name-text" title={shown}>{text}</span>
            : <button type="button" className="territory-card-player-name-text" title={shown} onClick={() => onOpen(line)}>{text}</button>}
        {line.admin && <AdminCrown/>}
        <StreakFlame days={streak}/>
    </span>
}

function Flag({code}: {code: string}) {
    const name = translator().country(code)
    return <span role="img" aria-label={name} title={name}><CountryFlag code={code}/></span>
}
