import {ReactNode, useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import {NO_TILE_DELTAS, TileDeltas} from "../domain/tileDeltas.ts";
import {TollStep} from "../domain/toll.ts";
import Leaderboard from "./Leaderboard.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import Modal from "./components/Modal.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";
import {CameraIcon, HomeIcon, InfoIcon, SpeakerIcon, SpeakerOffIcon, SwapIcon} from "./components/icons.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "./sound/SoundSettingsPanel.tsx";
import {AccountStore} from "./account/accountStore.ts";
import {useAccount} from "./account/useAccount.ts";
import AccountPanel from "./account/AccountPanel.tsx";
import DeleteAccountModal from "./account/DeleteAccountModal.tsx";
import {PlayerInfoBackend} from "../backends/player.ts";
import {youLabel} from "./youLabel.ts"
import "./Menu.css"

export type MenuTab = "board" | "you" | "more"

export type BoardPlaceProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tileDeltas?: TileDeltas,
    tilesCount: number,
    toll?: readonly TollStep[],
    anthem?: ReactNode,
    seasonEmails?: ReactNode,
}

export type YouPlaceProps = {
    account?: AccountStore,
    linkedMultiplier?: number,
    playerInfo?: PlayerInfoBackend,
    seasonEmails?: ReactNode,
}

export type MorePlaceProps = {
    sound?: SoundSettingsPanelProps,
    onTakePicture?: () => void,
    taking?: boolean,
}

export type MenuProps = BoardPlaceProps & YouPlaceProps & MorePlaceProps & {
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

    return <>
        {props.playing && <PlayingFor country={props.country}
                                      rank={rankOf(props.leaderboard, props.country)}
                                      changeRef={picker.opener}
                                      onChange={picker.open}/>}
        <Leaderboard data={props.leaderboard}
                     deltas={props.tileDeltas ?? NO_TILE_DELTAS}
                     tilesCount={props.tilesCount}
                     highlight={props.country}
                     toll={props.toll}
                     anthem={props.anthem}/>
        {props.seasonEmails}
    </>
}

export function YouPlace({account: store, linkedMultiplier, playerInfo, seasonEmails}: YouPlaceProps) {
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
                      emails={seasonEmails}
                      onDelete={() => setConfirmingDelete(true)}/>

        {confirmingDelete && <DeleteAccountModal
            linked={account.me.linked}
            busy={account.busy === "deleteAccount"}
            onConfirm={() => void deleteAccount()}
            onClose={() => setConfirmingDelete(false)}/>}
    </>
}

export function MorePlace({sound, onTakePicture, taking}: MorePlaceProps) {
    const [soundOpen, setSoundOpen] = useState(false)
    const [aboutOpen, setAboutOpen] = useState(false)
    const soundButton = useRef<HTMLButtonElement>(null)
    const cameFromSound = useRef(false)

    useEffect(() => {
        if (soundOpen || !cameFromSound.current) return
        cameFromSound.current = false
        soundButton.current?.focus()
    }, [soundOpen])

    if (soundOpen && sound) {
        return <MenuPanel title="Sound" onClose={() => setSoundOpen(false)}>
            <SoundSettingsPanel {...sound}/>
        </MenuPanel>
    }

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
            {sound && <button ref={soundButton}
                              type="button"
                              className="panel-box menu-tile"
                              onClick={() => {
                                  cameFromSound.current = true
                                  setSoundOpen(true)
                              }}>
                {sound.settings.enabled ? <SpeakerIcon size={22}/> : <SpeakerOffIcon size={22}/>}
                <span>Sound</span>
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
        </div>

        {aboutOpen && <Modal title="About ClickPlanet"
                             footer={<BuyMeACoffee/>}
                             onClose={() => setAboutOpen(false)}>
            <About/>
        </Modal>}
    </>
}
