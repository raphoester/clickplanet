import {useRef, useState} from 'react';
import {
    BankFullError,
    Bomber,
    BonusListener,
    BonusLostError,
    OwnershipsGetter,
    QuizMaster,
    Refiller,
    TileClicker,
    UpdatesListener,
} from "../../backends/backend.ts";
import BombNews from "../components/BombNews.tsx";
import NativeLandNote from "../components/NativeLandNote.tsx";
import Quiz from "../quiz/Quiz.tsx";
import {useQuiz} from "../quiz/useQuiz.ts";
import {ChatBackend} from "../../backends/chat.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu from "../Menu.tsx";
import BonusAward from "../components/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import Inventory from "../components/Inventory.tsx";
import SessionUnavailableModal from "../components/SessionUnavailableModal.tsx";
import VPNBlockedModal from "../components/VPNBlockedModal.tsx";
import CameraButton from "../share/CameraButton.tsx";
import SharePreview from "../share/SharePreview.tsx";
import {useSharePicture} from "../share/useSharePicture.ts";
import {shareStats} from "../../domain/shareCard.ts";
import {ClickBudgetSource, now as budgetNow, tokensAt} from "../../backends/clickBudget.ts";
import {useClickBudget} from './useClickBudget.ts';
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
import {useSound} from '../sound/useSound.ts';
import AnthemBar from "../anthem/AnthemBar.tsx";
import {useAnthem} from "../anthem/useAnthem.ts";
import {AccountStore} from "../account/accountStore.ts";
import {useAccount} from "../account/useAccount.ts";
import {PlayerInfoBackend, PlayerLine, PlayerTitle, PresenceBackend} from "../../backends/player.ts";
import PlayerCard from "../players/PlayerCard.tsx";
import TitleUnlocked from "../titles/TitleUnlocked.tsx";
import {usePresence} from "../players/usePresence.ts";
import {useRoster} from "../players/useRoster.ts";
import SignInPitchModal from "../account/SignInPitchModal.tsx";
import {LeaderboardEntry} from "../../domain/leaderboard.ts";
import "./Viewer.css"

const NO_LEADERBOARD: readonly LeaderboardEntry[] = []

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource?: ClickBudgetSource
    bonusListener?: BonusListener
    quizMaster?: QuizMaster
    bomber?: Bomber
    refiller?: Refiller
    chatBackend?: ChatBackend
    account?: AccountStore
    presence?: PresenceBackend
    playerInfo?: PlayerInfoBackend
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const sound = useSound()
    const account = useAccount(props.account)
    const username = account.kind === 'ready' ? account.username : undefined
    const color = account.kind === 'ready' ? account.color : undefined

    usePresence(props.presence, {countryCode: countryState.code, username, color})

    const quiz = useQuiz(props.quizMaster, countryState.code, sound.play)
    const [unlocked, setUnlocked] = useState<readonly PlayerTitle[]>([])
    const roster = useRoster(props.presence, (title) => setUnlocked((queue) => [...queue, title]))
    const [pitchOpen, setPitchOpen] = useState(false)
    const [openPlayer, setOpenPlayer] = useState<PlayerLine>()
    const onOpenPlayer = props.playerInfo ? setOpenPlayer : undefined
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
        charges,
        rules,
        bombArmed,
        toggleBomb,
        switches,
        toggleSwitch,
        lastBomb,
        dismissBomb,
        lastClear,
        dismissClear,
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

    const refiller = props.refiller
    const spendRefill = refiller && (() => {
        if (clickBudget && tokensAt(clickBudget, budgetNow()) >= clickBudget.capacity) return false
        refiller.useRefill(countryState.code).catch((e) => {
            if (e instanceof BankFullError || e instanceof BonusLostError) return
            console.error("could not use the refill", e)
        })
        return true
    })

    const anthem = useAnthem(status.state === 'ready' ? leaderboard : NO_LEADERBOARD, sound.settings)

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
                                                       countryName={countryState.name}
                                                       refusals={refusals}
                                                       onSignIn={guest ? () => setPitchOpen(true) : undefined}>
            {props.bonusListener && <Inventory charges={charges}
                                               rules={rules}
                                               switches={switches}
                                               onToggle={toggleSwitch}
                                               bombArmed={bombArmed}
                                               onToggleBomb={props.bomber ? toggleBomb : undefined}
                                               onUseRefill={spendRefill}/>}
        </ClickBudgetMeter>}

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
            onOpenPlayer={onOpenPlayer}
        />}

        {openPlayer && props.playerInfo && <PlayerCard key={openPlayer.name}
                                                       player={openPlayer}
                                                       backend={props.playerInfo}
                                                       onClose={() => setOpenPlayer(undefined)}/>}

        {unlocked.length > 0 && <TitleUnlocked key={unlocked[0].id}
                                               title={unlocked[0]}
                                               onWear={props.account?.wearTitle}
                                               onClose={() => setUnlocked((queue) => queue.slice(1))}/>}

        {award && <BonusAward reward={award} onDone={dismissAward}/>}

        {lastBomb && <BombNews
            key={lastBomb.id}
            drop={lastBomb.drop}
            land={lastBomb.land}
            lowered={quiz.state.phase !== 'idle'}
            onDone={dismissBomb}
        />}

        {lastClear && <NativeLandNote
            key={lastClear.id}
            ground={lastClear.ground}
            lowered={quiz.state.phase !== 'idle'}
            onDone={dismissClear}
        />}

        <Quiz state={quiz.state} onOpen={quiz.open} onAnswer={quiz.answer}/>

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
