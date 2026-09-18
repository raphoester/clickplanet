import {useRef, useState} from 'react';
import {Bomber, BonusListener, OwnershipsGetter, TileClicker, UpdatesListener} from "../../backends/backend.ts";
import BombNews from "../components/BombNews.tsx";
import {ChatBackend} from "../../backends/chat.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu from "../Menu.tsx";
import BonusAward from "../components/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import SessionUnavailableModal from "../components/SessionUnavailableModal.tsx";
import VPNBlockedModal from "../components/VPNBlockedModal.tsx";
import CameraButton from "../share/CameraButton.tsx";
import SharePreview from "../share/SharePreview.tsx";
import {useSharePicture} from "../share/useSharePicture.ts";
import {shareStats} from "../../domain/shareCard.ts";
import {ClickBudgetSource} from "../../backends/clickBudget.ts";
import {useClickBudget} from './useClickBudget.ts';
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
import {useSound} from '../sound/useSound.ts';
import AnthemBar from "../anthem/AnthemBar.tsx";
import {useAnthem} from "../anthem/useAnthem.ts";
import {AccountStore} from "../account/accountStore.ts";
import {useAccount} from "../account/useAccount.ts";
import {PlayerInfoBackend, PresenceBackend, PlayerLine} from "../../backends/player.ts";
import PlayerCard from "../players/PlayerCard.tsx";
import {useChatIdentity} from "../chat/useChatIdentity.ts";
import {usePresence} from "../players/usePresence.ts";
import {useRoster} from "../players/useRoster.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import "./Viewer.css"

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource?: ClickBudgetSource
    bonusListener?: BonusListener
    bomber?: Bomber
    chatBackend?: ChatBackend
    account?: AccountStore
    /** Absent — the fake backend without one — the menu lists no players. */
    presence?: PresenceBackend
    /** Absent, a name in the roster or the chat opens nothing. */
    playerInfo?: PlayerInfoBackend
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const sound = useSound()
    // The chat posts under the username, so it follows the account the menu shows.
    const account = useAccount(props.account)
    const username = account.kind === 'ready' ? account.username : undefined

    // Held here rather than in the chat: presence announces the same name, and
    // two copies of the hook would not hear of each other's change.
    const chatIdentity = useChatIdentity()
    usePresence(props.presence, {
        countryCode: countryState.code,
        guestName: chatIdentity.identity.name,
        username,
    })
    const roster = useRoster(props.presence)
    const [pitchOpen, setPitchOpen] = useState(false)
    // One card at a time, over the roster or the chat, whichever the name was clicked in.
    const [openPlayer, setOpenPlayer] = useState<PlayerLine>()
    const onOpenPlayer = props.playerInfo ? setOpenPlayer : undefined
    // A guest the server offers sign-in to. With sign-in off there is nothing to point at, so nothing is offered.
    const guest = account.kind === 'ready' && account.offered.length > 0 && account.me.linked.length === 0

    const {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        refusals,
        vpnBlocked,
        dismissVPNBlocked,
        sessionUnavailable,
        dismissSessionUnavailable,
        award,
        dismissAward,
        bonus,
        charges,
        bombArmed,
        toggleBomb,
        lastBomb,
        dismissBomb,
    } = useGlobe({
        container,
        tileClicker: props.tileClicker,
        ownershipsGetter: props.ownershipsGetter,
        updatesListener: props.updatesListener,
        bonusListener: props.bonusListener,
        bomber: props.bomber,
        playSound: sound.play,
        country: countryState,
    })

    // The camera lives out here rather than in the menu: the globe is what it
    // photographs, and the card over it is not in the picture.
    const anthem = useAnthem(leaderboard, sound.settings)

    const {shot, taking, take, discard} = useSharePicture(
        capture, shareStats(leaderboard, countryState))

    return <>
        <div ref={container} className="viewer-canvas"/>

        {status.state !== 'ready' && <StatusCard status={status}/>}

        {status.state === 'ready' && <Menu
            country={countryState}
            setCountry={handleSetCountry}
            leaderboard={leaderboard}
            tileDeltas={tileDeltas}
            tilesCount={tilesCount}
            sound={{settings: sound.settings, onChange: sound.setSettings, preview: sound.preview}}
            account={props.account}
            players={roster.kind === 'ready' ? roster.entries : undefined}
            onOpenPlayer={onOpenPlayer}
            linkedMultiplier={clickBudget?.linkedMultiplier}
        />}

        {status.state === 'ready' && <AnthemBar anthem={anthem}
                                                settings={sound.settings}
                                                onChange={sound.setSettings}/>}

        {status.state === 'ready' && <CameraButton busy={taking} onClick={take}/>}

        {shot && <SharePreview shot={shot}
                               stats={shareStats(leaderboard, countryState)}
                               onClose={discard}/>}

        {status.state === 'ready' && <ClickBudgetMeter budget={clickBudget}
                                                       bonus={bonus}
                                                       charges={charges}
                                                       bombArmed={bombArmed}
                                                       onToggleBomb={props.bomber ? toggleBomb : undefined}
                                                       countryName={countryState.name}
                                                       refusals={refusals}
                                                       onSignIn={guest ? () => setPitchOpen(true) : undefined}/>}

        {pitchOpen && guest && props.account && clickBudget?.linkedMultiplier && <SignInPitchModal
            state={account}
            store={props.account}
            multiplier={clickBudget.linkedMultiplier}
            onClose={() => setPitchOpen(false)}/>}

        {status.state === 'ready' && <ChatPanel
            backend={props.chatBackend}
            country={countryState}
            playSound={sound.play}
            username={username}
            identity={chatIdentity.identity}
            setName={chatIdentity.setName}
            onOpenPlayer={onOpenPlayer}
        />}

        {openPlayer && props.playerInfo && <PlayerCard key={`${openPlayer.name}#${openPlayer.tag}`}
                                                       player={openPlayer}
                                                       backend={props.playerInfo}
                                                       onClose={() => setOpenPlayer(undefined)}/>}

        {award && <BonusAward reward={award} onDone={dismissAward}/>}

        {lastBomb && <BombNews key={lastBomb.id} drop={lastBomb.drop} land={lastBomb.land} onDone={dismissBomb}/>}

        {vpnBlocked && <VPNBlockedModal onClose={dismissVPNBlocked}/>}

        {sessionUnavailable && <SessionUnavailableModal onClose={dismissSessionUnavailable}/>}
    </>
}

function StatusCard({status}: {status: GlobeStatus}) {
    return <div className="viewer-status">
        <div className="viewer-status-card" role="status">
            {status.state === 'loading'
                ? <LoadingCard territories={status.territories}/>
                : status.state === 'failed' && <>
                    <h3>The globe could not be loaded</h3>
                    <p>{status.message}</p>
                </>}
        </div>
    </div>
}

function LoadingCard({territories}: {territories: number | undefined}) {
    const percent = Math.round((territories ?? 0) * 100)
    return <>
        <div className="viewer-status-spinner"/>
        <h3>{territories === undefined ? "Loading the planet…" : "Loading territories…"}</h3>
        <div className="viewer-status-progress"
             role="progressbar"
             aria-label="Territories loaded"
             aria-valuemin={0}
             aria-valuemax={100}
             aria-valuenow={percent}>
            <div className="viewer-status-progress-fill" style={{width: `${percent}%`}}/>
        </div>
    </>
}
