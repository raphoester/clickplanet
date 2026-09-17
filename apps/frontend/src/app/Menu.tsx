import {useEffect, useId, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry, rankOf} from "../domain/leaderboard.ts";
import {NO_TILE_DELTAS, TileDeltas} from "../domain/tileDeltas.ts";
import Leaderboard from "./Leaderboard.tsx";
import MenuHeader from "./MenuHeader.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import CountryFlag from "./components/CountryFlag.tsx";
import Modal from "./components/Modal.tsx";
import BuyMeACoffee from "./components/BuyMeACoffee.tsx";
import {HomeIcon, InfoIcon, SpeakerIcon, SpeakerOffIcon, SwapIcon} from "./components/icons.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "./sound/SoundSettingsPanel.tsx";
import {opensFolded} from "./compact.ts";
import {AccountStore} from "./account/accountStore.ts";
import {useAccount} from "./account/useAccount.ts";
import AccountPanel, {AccountButton} from "./account/AccountPanel.tsx";
import DeleteAccountModal from "./account/DeleteAccountModal.tsx";
import {PlayerLine, RosterEntry} from "../backends/player.ts";
import PlayersPanel, {PlayersButton} from "./players/PlayersPanel.tsx";
import "./Menu.css"

export type MenuProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tileDeltas?: TileDeltas,
    tilesCount: number,
    /** Absent, the menu offers no sound settings. */
    sound?: SoundSettingsPanelProps,
    /** Absent, or with no provider offered, the menu offers no sign-in. */
    account?: AccountStore,
    /** Who is playing. Absent — no roster, or not read yet — the menu offers no list. */
    players?: readonly RosterEntry[],
    /** Absent, a name in the list opens nothing. */
    onOpenPlayer?: (player: PlayerLine) => void,
    /** What signing in multiplies the click allowance by, as the server said. Absent, the panel does not mention it. */
    linkedMultiplier?: number,
}

export default function Menu(props: MenuProps) {
    const [isOpen, setIsOpen] = useState(() => !opensFolded())
    const [pickingCountry, setPickingCountry] = useState(false)
    const [aboutOpen, setAboutOpen] = useState(false)
    const [soundOpen, setSoundOpen] = useState(false)
    const bodyId = useId()

    const cameFromChange = useRef(false)
    const changeButton = useRef<HTMLButtonElement>(null)

    const openPicker = () => {
        cameFromChange.current = true
        setPickingCountry(true)
    }

    useEffect(() => {
        if (pickingCountry || !cameFromChange.current) return
        cameFromChange.current = false
        changeButton.current?.focus()
    }, [pickingCountry])

    // Back from the sound panel lands on the button that opened it.
    const soundButton = useRef<HTMLButtonElement>(null)
    const cameFromSound = useRef(false)

    useEffect(() => {
        if (soundOpen || !cameFromSound.current) return
        cameFromSound.current = false
        soundButton.current?.focus()
    }, [soundOpen])

    const openSound = () => {
        cameFromSound.current = true
        setSoundOpen(true)
    }

    const account = useAccount(props.account)
    const [accountOpen, setAccountOpen] = useState(false)
    const [confirmingDelete, setConfirmingDelete] = useState(false)

    // Back from the account panel lands on the button that opened it.
    const accountButton = useRef<HTMLButtonElement>(null)
    const cameFromAccount = useRef(false)

    useEffect(() => {
        if (accountOpen || !cameFromAccount.current) return
        cameFromAccount.current = false
        accountButton.current?.focus()
    }, [accountOpen])

    const openAccount = () => {
        cameFromAccount.current = true
        setAccountOpen(true)
    }

    const [playersOpen, setPlayersOpen] = useState(false)

    // Back from the players panel lands on the button that opened it.
    const playersButton = useRef<HTMLButtonElement>(null)
    const cameFromPlayers = useRef(false)

    useEffect(() => {
        if (playersOpen || !cameFromPlayers.current) return
        cameFromPlayers.current = false
        playersButton.current?.focus()
    }, [playersOpen])

    const openPlayers = () => {
        cameFromPlayers.current = true
        setPlayersOpen(true)
    }

    const deleteAccount = async () => {
        await props.account?.deleteAccount()
        setConfirmingDelete(false)
    }

    const pickCountry = (country: Country) => {
        props.setCountry(country)
        setPickingCountry(false)
    }

    return <>
        <div className="menu">
            <MenuHeader country={props.country}
                        rank={rankOf(props.leaderboard, props.country)}
                        isOpen={isOpen}
                        onToggle={() => setIsOpen(!isOpen)}
                        bodyId={bodyId}/>

            {isOpen && <div className="menu-body" id={bodyId}>
                {pickingCountry
                    ? <MenuPanel title="Change country" onClose={() => setPickingCountry(false)}>
                        <CountryPicker country={props.country} setCountry={pickCountry}/>
                    </MenuPanel>
                    : soundOpen && props.sound
                    ? <MenuPanel title="Sound" onClose={() => setSoundOpen(false)}>
                        <SoundSettingsPanel {...props.sound}/>
                    </MenuPanel>
                    : accountOpen && props.account && account.kind === "ready"
                    ? <MenuPanel title="Account" onClose={() => setAccountOpen(false)}>
                        <AccountPanel state={account}
                                      store={props.account}
                                      linkedMultiplier={props.linkedMultiplier}
                                      onDelete={() => setConfirmingDelete(true)}/>
                    </MenuPanel>
                    : playersOpen && props.players
                    ? <MenuPanel title="Players online" onClose={() => setPlayersOpen(false)}>
                        <PlayersPanel entries={props.players} onOpenPlayer={props.onOpenPlayer}/>
                    </MenuPanel>
                    : <>
                        <div className="menu-playing">
                            <span className="menu-label">You’re playing for</span>
                            <div className="menu-playing-row">
                                <span className="menu-playing-name">
                                    <CountryFlag code={props.country.code}/>
                                    {props.country.name}
                                </span>
                                <button ref={changeButton}
                                        type="button"
                                        className="button button-mini menu-change"
                                        onClick={openPicker}>
                                    <SwapIcon/>
                                    <span>Change</span>
                                </button>
                            </div>
                        </div>

                        <Leaderboard data={props.leaderboard}
                                     deltas={props.tileDeltas ?? NO_TILE_DELTAS}
                                     tilesCount={props.tilesCount}
                                     highlight={props.country}/>

                        <div className="menu-actions">
                            <a href="/#home"
                               className="button button-ghost menu-icon"
                               aria-label="Home"
                               title="Home">
                                <HomeIcon size={26}/>
                            </a>
                            {account.kind === "ready" && <AccountButton state={account}
                                                                        buttonRef={accountButton}
                                                                        onOpen={openAccount}/>}
                            {props.players && <PlayersButton entries={props.players}
                                                             buttonRef={playersButton}
                                                             onOpen={openPlayers}/>}
                            <button type="button"
                                    className="button button-ghost menu-icon"
                                    aria-label="About"
                                    title="About"
                                    onClick={() => setAboutOpen(true)}>
                                <InfoIcon size={26}/>
                            </button>
                            {props.sound && <button ref={soundButton}
                                                    type="button"
                                                    className="button button-ghost menu-sound"
                                                    aria-label="Sound settings"
                                                    onClick={openSound}>
                                {props.sound.settings.enabled ? <SpeakerIcon size={26}/> : <SpeakerOffIcon size={26}/>}
                            </button>}
                        </div>
                    </>}
            </div>}
        </div>

        {confirmingDelete && account.kind === "ready" && <DeleteAccountModal
            linked={account.me.linked}
            busy={account.busy === "deleteAccount"}
            onConfirm={() => void deleteAccount()}
            onClose={() => setConfirmingDelete(false)}/>}

        {aboutOpen && <Modal title="About ClickPlanet"
                             footer={<BuyMeACoffee/>}
                             onClose={() => setAboutOpen(false)}>
            <About/>
        </Modal>}
    </>
}
