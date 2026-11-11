import {ReactNode, useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import {countryLines, CountryOrder, shownOrder} from "../domain/race.ts";
import {NO_TILE_DELTAS, TileDeltas} from "../domain/tileDeltas.ts";
import {Race} from "../backends/standings.ts";
import {Allegiance, Lock, lockOf, minutesLeftOf} from "../domain/allegiance.ts";
import {describeEnd} from "../domain/endTime.ts";
import Leaderboard from "./Leaderboard.tsx";
import CountryPicker from "./CountryPicker.tsx";
import {useStanding} from "./knockout/standingStore.ts";
import MenuPanel from "./components/MenuPanel.tsx";
import {useEdge} from "./components/useEdge.ts";
import HomeLogo from "./components/HomeLogo.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import {HideIcon, LockIcon, SettingsIcon, ShareIcon, SwapIcon, TargetIcon, TrophyIcon, UserIcon, UsersIcon} from "./components/icons.tsx";
import SettingsPlace, {SettingsPlaceProps} from "./settings/SettingsPlace.tsx";
import {AccountStore} from "./account/accountStore.ts";
import {useAccount} from "./account/useAccount.ts";
import AccountPanel from "./account/AccountPanel.tsx";
import ProfileModal from "./account/ProfileModal.tsx";
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
import {ListenForClicks} from "./clicks/acceptedClicks.ts"
import {useNow} from "./season/useNow.ts"
import {HallOfFameButton, HallOfFameSeasons} from "./season/HallOfFame.tsx"
import {MenuTab, useScreen} from "./hud/screenStore.ts"
import {useRace} from "./standings/raceStore.ts"
import "./Menu.css"

export type BoardPlaceProps = {
    country: Country,
    setCountry: (country: Country) => Promise<void> | void,
    allegiance?: Allegiance,
    leaderboard: LeaderboardEntry[],
    tileDeltas?: TileDeltas,
    tilesCount: number,
    anthem?: ReactNode,
    standings?: BoardStandings,
    countryOrder?: CountryOrder,
    onCountryOrder?: (order: CountryOrder) => void,
    hallOfFame?: HallOfFameSeasons,
}

export type YouPlaceProps = {
    account: AccountStore,
    linkedMultiplier?: number,
    playerInfo: PlayerInfoBackend,
    listenForClicks?: ListenForClicks,
    clips: ClipsNameBackend,
}

export type MenuProps = BoardPlaceProps & YouPlaceProps & SettingsPlaceProps & Omit<SharePlaceProps, "stats"> & {
    missions?: MissionsPlaceProps,
    missionsNews?: number,
    invite?: InviteLink,
}

const OPEN_KEY = "clickplanet-menu-open"

// Where the menu ends: what is centred at the top centres in the room right of it.
export const MENU_RIGHT = "--menu-right"

type Place = {id: MenuTab, label: string, title: string, icon: ReactNode, news?: number}

export default function Menu(props: MenuProps) {
    const tab = useScreen((screen) => screen.menuTab)
    const setTab = useScreen((screen) => screen.setMenuTab)
    const race = useRace((standings) => standings.race)
    const account = useAccount(props.account)
    const tabsId = useId()
    const panelId = useId()
    const [open, setOpen] = useMenuOpen()
    const [profileOpen, setProfileOpen] = useState(false)
    const picker = usePicker()
    const [shell, setShell] = useState<HTMLElement | null>(null)
    useEdge(shell, MENU_RIGHT, "right")

    const places: Place[] = [
        {id: "board", label: "Board", title: "Leaderboard", icon: <TrophyIcon/>},
        ...props.missions ? [{id: "missions" as const, label: "Missions", title: "Missions", icon: <TargetIcon/>, news: props.missionsNews}] : [],
        props.invite
            ? {id: "invite", label: "Invite", title: "Invite friends", icon: <UsersIcon/>}
            : {id: "share", label: "Share", title: "Share", icon: <ShareIcon size={22}/>},
        {id: "settings", label: "Settings", title: "Settings", icon: <SettingsIcon/>},
    ]
    const shown = places.find((p) => p.id === tab) ?? places[0]
    const linked = account.kind === "ready" && account.me.linked.length > 0

    // A place pressed again folds the panel away, as a tab does on a phone.
    const pickPlace = (next: MenuTab) => {
        picker.close()
        if (open && next === shown.id) {
            setOpen(false)
            return
        }
        setTab(next)
        setOpen(true)
    }
    const change = () => {
        setOpen(true)
        picker.open()
    }

    return <div ref={setShell} className="menu-shell">
        <nav className="menu-rail panel" aria-label="Places">
            <h1 className="sr-only">ClickPlanet</h1>
            <HomeLogo size={40} className="menu-rail-logo"/>
            <div className="menu-rail-tabs" role="tablist" aria-label="Menu" aria-orientation="vertical">
                {places.map((p) => <button key={p.id}
                                           type="button"
                                           role="tab"
                                           id={`${tabsId}-${p.id}`}
                                           aria-selected={open && shown.id === p.id}
                                           aria-controls={open ? panelId : undefined}
                                           className={open && shown.id === p.id ? "menu-rail-item menu-rail-item--on" : "menu-rail-item"}
                                           aria-label={p.news ? `${p.label}, ${p.news} new` : undefined}
                                           onClick={() => pickPlace(p.id)}>
                    {p.icon}
                    <span className="menu-rail-label">{p.label}</span>
                    {!!p.news && <span className="chip menu-tab-badge" key={p.news} aria-hidden="true">{p.news}</span>}
                </button>)}
            </div>
            {account.kind === "ready" && <button type="button"
                                                 className="menu-rail-item menu-rail-you"
                                                 aria-haspopup="dialog"
                                                 onClick={() => setProfileOpen(true)}>
                <UserIcon/>
                <span className="menu-rail-label">{youLabel(linked)}</span>
            </button>}
        </nav>

        {open
            ? <aside className="menu panel" id={panelId} aria-label={shown.title}>
                {picker.picking
                    ? <MenuPanel title="Change country" onClose={picker.close}>
                        <CountryPicker country={props.country} allegiance={props.allegiance} setCountry={async (country) => {
                            await props.setCountry(country)
                            picker.close()
                        }}/>
                    </MenuPanel>
                    : <>
                        <div className="menu-header">
                            {shown.id === "board" && props.standings
                                ? <BoardHeading view={props.standings.view}
                                                onView={props.standings.onView}
                                                country={props.country}/>
                                : <h2 className="menu-title">{shown.title}</h2>}
                            {shown.id === "board" && props.hallOfFame && <HallOfFameButton {...props.hallOfFame} icon/>}
                            <button type="button"
                                    className="icon-button menu-hide"
                                    aria-label="Hide the panel"
                                    aria-expanded={true}
                                    aria-controls={panelId}
                                    onClick={() => setOpen(false)}>
                                <HideIcon/>
                            </button>
                        </div>

                        <PlayingFor country={props.country}
                                    allegiance={props.allegiance}
                                    rank={rankShown(props, race)}
                                    changeRef={picker.opener}
                                    onChange={change}/>

                        <div className="menu-body"
                             role="tabpanel"
                             aria-labelledby={`${tabsId}-${shown.id}`}>
                            {shown.id === "board" && <BoardPlace {...props} hallOfFame={undefined} headless={props.standings !== undefined}/>}
                            {shown.id === "missions" && props.missions && <MissionsPlace {...props.missions}/>}
                            {shown.id === "share" && <SharePlace stats={shareStats(linesShown(props, race), props.country)}
                                                                 onTakePicture={props.onTakePicture}
                                                                 taking={props.taking}/>}
                            {shown.id === "invite" && props.invite && <InviteFriends invite={props.invite}/>}
                            {shown.id === "settings" && <SettingsPlace {...props}/>}
                        </div>
                    </>}
            </aside>
            : <PlayingFor country={props.country}
                          allegiance={props.allegiance}
                          rank={rankShown(props, race)}
                          changeRef={picker.opener}
                          onChange={change}
                          pill/>}

        {profileOpen && <ProfileModal account={props.account}
                                      linkedMultiplier={props.linkedMultiplier}
                                      playerInfo={props.playerInfo}
                                      listenForClicks={props.listenForClicks}
                                      clips={props.clips}
                                      country={props.country}
                                      onClose={() => setProfileOpen(false)}/>}
    </div>
}

// The panel stays as the player left it, folded or open, from one visit to the next.
function useMenuOpen(): [boolean, (open: boolean) => void] {
    const [open, setOpen] = useState(() => {
        try {
            return localStorage.getItem(OPEN_KEY) !== "false"
        } catch {
            return true
        }
    })
    return [open, (next: boolean) => {
        setOpen(next)
        try {
            localStorage.setItem(OPEN_KEY, String(next))
        } catch {
            // A browser that keeps nothing folds the panel for this visit only.
        }
    }]
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
    // The panel folded away: the flag stays in sight, beside the rail.
    pill?: boolean,
}

function PlayingFor({country, allegiance, rank, changeRef, onChange, slim, pill}: PlayingForProps) {
    const className = slim ? "menu-playing menu-playing--slim"
        : pill ? "menu-playing menu-playing--pill panel"
            : "menu-playing panel-box"
    return <div className={className}>
        <div className="menu-playing-what">
            <span className="menu-label">{slim || pill ? "Playing for" : "You’re playing for"}</span>
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
    const standing = useStanding((store) => store.standing)
    const lock = lockOf(allegiance, now, standing)

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
    const countries = <Leaderboard data={props.leaderboard}
                                   deltas={props.tileDeltas ?? NO_TILE_DELTAS}
                                   tilesCount={props.tilesCount}
                                   highlight={props.country}
                                   anthem={props.anthem}
                                   order={props.countryOrder}
                                   onOrder={props.onCountryOrder}/>

    const hallOfFame = props.hallOfFame && <HallOfFameButton {...props.hallOfFame}/>

    if (props.standings) {
        return <>
            {hallOfFame}
            <BoardViews {...props.standings}
                        country={props.country}
                        countries={countries}
                        headless={props.headless}/>
        </>
    }
    return <>
        {hallOfFame}
        {countries}
    </>
}

export function BoardSheet({onClose, ...props}: BoardPlaceProps & {onClose: () => void}) {
    const picker = usePicker()
    const race = useRace((standings) => standings.race)
    const standings = props.standings
    const head = standings && !picker.picking
        ? <BoardHeading view={standings.view} onView={standings.onView} country={props.country}/>
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
                            rank={rankShown(props, race)}
                            changeRef={picker.opener}
                            onChange={picker.open}
                            slim/>
                <BoardPlace {...props} headless={head !== undefined}/>
            </>}
    </Sheet>
}

function linesShown(props: BoardPlaceProps, race: Race | undefined) {
    return countryLines(props.leaderboard, race, shownOrder(race, props.countryOrder))
}

function rankShown(props: BoardPlaceProps, race: Race | undefined): number | null {
    return rankOf(linesShown(props, race), props.country)
}

export function YouPlace({account: store, linkedMultiplier, playerInfo, listenForClicks, clips}: YouPlaceProps) {
    const account = useAccount(store)
    const [confirmingDelete, setConfirmingDelete] = useState(false)

    if (account.kind !== "ready") return null

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
