import {ReactNode, useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import {countryLines, CountryOrder, shownOrder} from "../domain/race.ts";
import {NO_TILE_DELTAS, TileDeltas} from "../domain/tileDeltas.ts";
import {Race} from "../backends/standings.ts";
import {Allegiance, Lock, lockOf, minutesLeftOf} from "../domain/allegiance.ts";
import {describeEnd} from "../domain/endTime.ts";
import Leaderboard, {OrderSwitch} from "./Leaderboard.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import {LockIcon, SwapIcon} from "./components/icons.tsx";
import SettingsPlace, {SettingsPlaceProps} from "./settings/SettingsPlace.tsx";
import {AccountStore} from "./account/accountStore.ts";
import {useAccount} from "./account/useAccount.ts";
import AccountPanel from "./account/AccountPanel.tsx";
import MissionsPlace, {MissionsPlaceProps} from "./missions/MissionsPlace.tsx";
import DeleteAccountModal from "./account/DeleteAccountModal.tsx";
import {PlayerInfoBackend} from "../backends/player.ts";
import {ClipsNameBackend} from "../backends/clips.ts";
import {shareStats} from "../domain/shareCard.ts";
import SharePlace, {SharePlaceProps} from "./share/SharePlace.tsx";
import InviteFriends from "./share/InviteFriends.tsx";
import {InviteLink} from "../backends/crazyGames.ts";
import {youLabel} from "./youLabel.ts"
import BoardViews, {BoardHeading, BoardStandings} from "./standings/BoardViews.tsx"
import Sheet from "./hud/Sheet.tsx"
import {ListenForClicks} from "./viewer/acceptedClicks.ts"
import {useNow} from "./season/useNow.ts"
import {HallOfFameButton, HallOfFameSeasons} from "./season/HallOfFame.tsx"
import "./Menu.css"

export type MenuTab = "board" | "missions" | "you" | "share" | "invite" | "settings"

export type BoardPlaceProps = {
    country: Country,
    setCountry: (country: Country) => Promise<void> | void,
    allegiance?: Allegiance,
    leaderboard: LeaderboardEntry[],
    tileDeltas?: TileDeltas,
    tilesCount: number,
    anthem?: ReactNode,
    standings?: BoardStandings,
    race?: Race,
    countryOrder?: CountryOrder,
    onCountryOrder?: (order: CountryOrder) => void,
    guided?: boolean,
    onGuided?: () => void,
    hallOfFame?: HallOfFameSeasons,
}

export type YouPlaceProps = {
    account?: AccountStore,
    linkedMultiplier?: number,
    playerInfo?: PlayerInfoBackend,
    listenForClicks?: ListenForClicks,
    clips?: ClipsNameBackend,
}

export type MenuProps = BoardPlaceProps & YouPlaceProps & SettingsPlaceProps & Omit<SharePlaceProps, "stats"> & {
    missions?: MissionsPlaceProps,
    missionsNews?: number,
    invite?: InviteLink,
    tab?: MenuTab,
    onTab?: (tab: MenuTab) => void,
}

export default function Menu(props: MenuProps) {
    const [ownTab, setOwnTab] = useState<MenuTab>("board")
    const tab = props.tab ?? ownTab
    const pickTab = (next: MenuTab) => props.onTab ? props.onTab(next) : setOwnTab(next)
    const account = useAccount(props.account)
    const tabsId = useId()
    const picker = usePicker()

    const tabs: {id: MenuTab, label: string, news?: number}[] = [
        {id: "board", label: "Board"},
        ...props.missions ? [{id: "missions" as const, label: "Missions", news: props.missionsNews}] : [],
        ...account.kind === "ready" ? [{id: "you" as const, label: youLabel(account.me.linked.length > 0)}] : [],
        props.invite ? {id: "invite", label: "Invite"} : {id: "share", label: "Share"},
        {id: "settings", label: "Settings"},
    ]
    const shown = tabs.some((t) => t.id === tab) ? tab : "board"

    return <aside className="menu panel" aria-label="Menu">
        <div className="menu-header">
            <img alt="ClickPlanet logo"
                 src="/static/logo.svg"
                 className="menu-header-logo"
                 width="40px"
                 height="40px"/>
            <h1>ClickPlanet</h1>
        </div>

        <PlayingFor country={props.country}
                    allegiance={props.allegiance}
                    rank={rankShown(props)}
                    changeRef={picker.opener}
                    onChange={picker.open}/>

        {picker.picking
            ? <MenuPanel title="Change country" onClose={picker.close}>
                <CountryPicker country={props.country} allegiance={props.allegiance} setCountry={async (country) => {
                    await props.setCountry(country)
                    picker.close()
                }}/>
            </MenuPanel>
            : <>
                <div className="menu-tabs" role="tablist" aria-label="Menu">
                    {tabs.map((t) => <button key={t.id}
                                             type="button"
                                             role="tab"
                                             id={`${tabsId}-${t.id}`}
                                             aria-selected={shown === t.id}
                                             aria-controls={`${tabsId}-panel`}
                                             className={shown === t.id ? "menu-tab menu-tab--on" : "menu-tab"}
                                             aria-label={t.news ? `${t.label}, ${t.news} new` : undefined}
                                             onClick={() => pickTab(t.id)}>
                        {t.label}
                        {!!t.news && <span className="chip menu-tab-badge" key={t.news} aria-hidden="true">{t.news}</span>}
                    </button>)}
                </div>
                <div className="menu-body"
                     role="tabpanel"
                     id={`${tabsId}-panel`}
                     aria-labelledby={`${tabsId}-${shown}`}>
                    {shown === "board" && <BoardPlace {...props}/>}
                    {shown === "missions" && props.missions && <MissionsPlace {...props.missions}/>}
                    {shown === "you" && <YouPlace {...props}/>}
                    {shown === "share" && <SharePlace stats={shareStats(linesShown(props), props.country)}
                                                      onTakePicture={props.onTakePicture}
                                                      taking={props.taking}/>}
                    {shown === "invite" && props.invite && <InviteFriends country={props.country} invite={props.invite}/>}
                    {shown === "settings" && <SettingsPlace {...props}/>}
                </div>
            </>}
    </aside>
}

function usePicker() {
    const [picking, setPicking] = useState(false)
    const opener = useRef<HTMLButtonElement>(null)
    const cameFrom = useRef(false)

    useEffect(() => {
        if (picking || !cameFrom.current) return
        cameFrom.current = false
        opener.current?.focus()
    }, [picking])

    return {
        picking,
        opener,
        open: () => {
            cameFrom.current = true
            setPicking(true)
        },
        close: () => setPicking(false),
    }
}

type PlayingForProps = {
    country: Country,
    allegiance?: Allegiance,
    rank: number | null,
    changeRef: React.Ref<HTMLButtonElement>,
    onChange: () => void,
    slim?: boolean,
}

function PlayingFor({country, allegiance, rank, changeRef, onChange, slim}: PlayingForProps) {
    return <div className={slim ? "menu-playing menu-playing--slim" : "menu-playing panel-box"}>
        <div className="menu-playing-what">
            <span className="menu-label">{slim ? "Playing for" : "You’re playing for"}</span>
            <span className="menu-playing-name">
                <CountryFlag code={country.code}/>
                <span className="menu-playing-country">{country.name}</span>
                <span className="menu-rank-value" aria-label={rank === null ? "No rank yet" : `Rank ${rank}`}>
                    {rank === null ? "—" : `#${rank}`}
                </span>
            </span>
        </div>
        <ChangeButton allegiance={allegiance} changeRef={changeRef} onChange={onChange}/>
    </div>
}

function ChangeButton({allegiance, changeRef, onChange}: Pick<PlayingForProps, "allegiance" | "changeRef" | "onChange">) {
    useNow()
    const now = Date.now()
    const lock = lockOf(allegiance, now)

    return <button ref={changeRef}
                   type="button"
                   className={lock === undefined ? "button button-mini button-secondary menu-change" : "button button-mini button-secondary menu-change menu-change--locked"}
                   aria-label={lock === undefined ? undefined : lockedChangeLabelOf(lock, now)}
                   onClick={onChange}>
        {lock === undefined
            ? <><SwapIcon/><span>Change</span></>
            : <><LockIcon/><span>{lockedChangeTextOf(lock, now)}</span></>}
    </button>
}

function lockedChangeLabelOf(lock: Lock, now: number): string {
    switch (lock.kind) {
        case "rules":
            return "Change country, locked for now"
        case "wait":
            return `Change country, next change at ${describeEnd(lock.until, now)}`
    }
}

function lockedChangeTextOf(lock: Lock, now: number): string {
    switch (lock.kind) {
        case "rules":
            return "Locked"
        case "wait":
            return `${minutesLeftOf(lock.until, now)} min`
    }
}

export function BoardPlace(props: BoardPlaceProps & {headless?: boolean}) {
    const order = orderOf(props)
    const countries = <Leaderboard data={props.leaderboard}
                                   deltas={props.tileDeltas ?? NO_TILE_DELTAS}
                                   tilesCount={props.tilesCount}
                                   highlight={props.country}
                                   anthem={props.anthem}
                                   race={props.race}
                                   order={props.countryOrder}
                                   guided={props.guided}
                                   onGuided={props.onGuided}/>

    const hallOfFame = props.hallOfFame && <HallOfFameButton {...props.hallOfFame}/>

    if (props.standings) {
        return <>
            {hallOfFame}
            <BoardViews {...props.standings}
                        country={props.country}
                        countries={countries}
                        order={order}
                        headless={props.headless}/>
        </>
    }
    return <>
        {hallOfFame}
        {order && <div className="board-heading">{order}</div>}
        {countries}
    </>
}

export function BoardSheet({onClose, ...props}: BoardPlaceProps & {onClose: () => void}) {
    const picker = usePicker()
    const standings = props.standings
    const head = standings && !picker.picking
        ? <BoardHeading view={standings.view} onView={standings.onView} country={props.country} order={orderOf(props)}/>
        : undefined

    return <Sheet title="Leaderboard" head={head} className="board-sheet" onClose={onClose}>
        {picker.picking
            ? <MenuPanel title="Change country" onClose={picker.close}>
                <CountryPicker country={props.country} allegiance={props.allegiance} setCountry={async (country) => {
                    await props.setCountry(country)
                    picker.close()
                }}/>
            </MenuPanel>
            : <>
                <PlayingFor country={props.country}
                            allegiance={props.allegiance}
                            rank={rankShown(props)}
                            changeRef={picker.opener}
                            onChange={picker.open}
                            slim/>
                <BoardPlace {...props} headless={head !== undefined}/>
            </>}
    </Sheet>
}

function orderOf(props: BoardPlaceProps): ReactNode {
    const onOrder = props.onCountryOrder
    if (!props.race || !onOrder) return undefined
    return <OrderSwitch order={shownOrder(props.race, props.countryOrder)} onOrder={onOrder}/>
}

function linesShown(props: BoardPlaceProps) {
    return countryLines(props.leaderboard, props.race, shownOrder(props.race, props.countryOrder))
}

function rankShown(props: BoardPlaceProps): number | null {
    return rankOf(linesShown(props), props.country)
}

export function YouPlace({account: store, linkedMultiplier, playerInfo, listenForClicks, clips}: YouPlaceProps) {
    const account = useAccount(store)
    const [confirmingDelete, setConfirmingDelete] = useState(false)

    if (!store || account.kind !== "ready") return null

    const deleteAccount = async () => {
        await store.deleteAccount()
        setConfirmingDelete(false)
    }

    return <>
        <AccountPanel state={account}
                      store={store}
                      linkedMultiplier={linkedMultiplier}
                      playerInfo={playerInfo}
                      listenForClicks={listenForClicks}
                      clips={clips}
                      onDelete={() => setConfirmingDelete(true)}/>

        {confirmingDelete && <DeleteAccountModal
            linked={account.me.linked}
            busy={account.busy === "deleteAccount"}
            onConfirm={() => void deleteAccount()}
            onClose={() => setConfirmingDelete(false)}/>}
    </>
}
