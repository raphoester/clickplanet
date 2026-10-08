import {ReactNode, useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import {CountryOrder} from "../domain/race.ts";
import {NO_TILE_DELTAS, TileDeltas} from "../domain/tileDeltas.ts";
import {Race} from "../backends/standings.ts";
import {TollStep} from "../domain/toll.ts";
import Leaderboard from "./Leaderboard.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import Modal from "./components/Modal.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";
import {CameraIcon, DiscordIcon, HomeIcon, InfoIcon, InstagramIcon, SwapIcon, TikTokIcon} from "./components/icons.tsx";
import SettingsPlace, {SettingsPlaceProps} from "./settings/SettingsPlace.tsx";
import {AccountStore} from "./account/accountStore.ts";
import {useAccount} from "./account/useAccount.ts";
import AccountPanel from "./account/AccountPanel.tsx";
import DeleteAccountModal from "./account/DeleteAccountModal.tsx";
import {PlayerInfoBackend} from "../backends/player.ts";
import {youLabel} from "./youLabel.ts"
import {DISCORD_INVITE, INSTAGRAM_PROFILE, TIKTOK_PROFILE} from "../links.ts"
import BoardViews, {BoardStandings} from "./standings/BoardViews.tsx"
import {ListenForClicks} from "./viewer/acceptedClicks.ts"
import "./Menu.css"

export type MenuTab = "board" | "you" | "settings" | "more"

export type BoardPlaceProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tileDeltas?: TileDeltas,
    tilesCount: number,
    toll?: readonly TollStep[],
    anthem?: ReactNode,
    standings?: BoardStandings,
    race?: Race,
    countryOrder?: CountryOrder,
    onCountryOrder?: (order: CountryOrder) => void,
    guided?: boolean,
    onGuided?: () => void,
}

export type YouPlaceProps = {
    account?: AccountStore,
    linkedMultiplier?: number,
    playerInfo?: PlayerInfoBackend,
    listenForClicks?: ListenForClicks,
}

export type MorePlaceProps = {
    onTakePicture?: () => void,
    taking?: boolean,
}

export type MenuProps = BoardPlaceProps & YouPlaceProps & SettingsPlaceProps & MorePlaceProps & {
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

    const tabs: {id: MenuTab, label: string}[] = [
        {id: "board", label: "Board"},
        ...account.kind === "ready" ? [{id: "you" as const, label: youLabel(account.me.linked.length > 0)}] : [],
        ...props.display || props.sound ? [{id: "settings" as const, label: "Settings"}] : [],
        {id: "more", label: "More"},
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
                    rank={rankOf(props.leaderboard, props.country)}
                    changeRef={picker.opener}
                    onChange={picker.open}/>

        {picker.picking
            ? <MenuPanel title="Change country" onClose={picker.close}>
                <CountryPicker country={props.country} setCountry={(country) => {
                    props.setCountry(country)
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
                                             onClick={() => pickTab(t.id)}>
                        {t.label}
                    </button>)}
                </div>
                <div className="menu-body"
                     role="tabpanel"
                     id={`${tabsId}-panel`}
                     aria-labelledby={`${tabsId}-${shown}`}>
                    {shown === "board" && <BoardPlace {...props} playing={false}/>}
                    {shown === "you" && <YouPlace {...props}/>}
                    {shown === "settings" && <SettingsPlace {...props}/>}
                    {shown === "more" && <MorePlace {...props}/>}
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
    rank: number | null,
    changeRef: React.Ref<HTMLButtonElement>,
    onChange: () => void,
}

function PlayingFor({country, rank, changeRef, onChange}: PlayingForProps) {
    return <div className="menu-playing panel-box">
        <div className="menu-playing-what">
            <span className="menu-label">You’re playing for</span>
            <span className="menu-playing-name">
                <CountryFlag code={country.code}/>
                <span className="menu-playing-country">{country.name}</span>
                <span className="menu-rank-value" aria-label={rank === null ? "No rank yet" : `Rank ${rank}`}>
                    {rank === null ? "—" : `#${rank}`}
                </span>
            </span>
        </div>
        <button ref={changeRef}
                type="button"
                className="button button-mini button-secondary menu-change"
                onClick={onChange}>
            <SwapIcon/>
            <span>Change</span>
        </button>
    </div>
}

export function BoardPlace(props: BoardPlaceProps & {playing: boolean}) {
    const picker = usePicker()

    if (props.playing && picker.picking) {
        return <MenuPanel title="Change country" onClose={picker.close}>
            <CountryPicker country={props.country} setCountry={(country) => {
                props.setCountry(country)
                picker.close()
            }}/>
        </MenuPanel>
    }

    const countries = <Leaderboard data={props.leaderboard}
                                   deltas={props.tileDeltas ?? NO_TILE_DELTAS}
                                   tilesCount={props.tilesCount}
                                   highlight={props.country}
                                   toll={props.toll}
                                   anthem={props.anthem}
                                   race={props.race}
                                   order={props.countryOrder}
                                   onOrder={props.onCountryOrder}
                                   guided={props.guided}
                                   onGuided={props.onGuided}/>

    return <>
        {props.playing && <PlayingFor country={props.country}
                                      rank={rankOf(props.leaderboard, props.country)}
                                      changeRef={picker.opener}
                                      onChange={picker.open}/>}
        {props.standings
            ? <BoardViews {...props.standings} country={props.country} countries={countries}/>
            : countries}
    </>
}

export function YouPlace({account: store, linkedMultiplier, playerInfo, listenForClicks}: YouPlaceProps) {
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
                      onDelete={() => setConfirmingDelete(true)}/>

        {confirmingDelete && <DeleteAccountModal
            linked={account.me.linked}
            busy={account.busy === "deleteAccount"}
            onConfirm={() => void deleteAccount()}
            onClose={() => setConfirmingDelete(false)}/>}
    </>
}

export function MorePlace({onTakePicture, taking}: MorePlaceProps) {
    const [aboutOpen, setAboutOpen] = useState(false)

    return <>
        <div className="menu-tiles">
            {onTakePicture && <button type="button"
                                      className="panel-box menu-tile"
                                      aria-busy={taking}
                                      disabled={taking}
                                      onClick={onTakePicture}>
                <CameraIcon size={22}/>
                <span>Take a picture</span>
            </button>}
            <button type="button"
                    className="panel-box menu-tile"
                    onClick={() => setAboutOpen(true)}>
                <InfoIcon size={22}/>
                <span>About</span>
            </button>
            <a href="/#home" className="panel-box menu-tile">
                <HomeIcon size={22}/>
                <span>Home page</span>
            </a>
            <a href={DISCORD_INVITE}
               target="_blank"
               rel="noopener noreferrer"
               className="panel-box menu-tile">
                <DiscordIcon size={22}/>
                <span>Discord</span>
            </a>
            <a href={TIKTOK_PROFILE}
               target="_blank"
               rel="noopener noreferrer"
               className="panel-box menu-tile">
                <TikTokIcon size={22}/>
                <span>TikTok</span>
            </a>
            <a href={INSTAGRAM_PROFILE}
               target="_blank"
               rel="noopener noreferrer"
               className="panel-box menu-tile">
                <InstagramIcon size={22}/>
                <span>Instagram</span>
            </a>
        </div>

        {aboutOpen && <Modal title="About ClickPlanet"
                             footer={<BuyMeACoffee/>}
                             onClose={() => setAboutOpen(false)}>
            <About/>
        </Modal>}
    </>
}
