import {useRef} from 'react';
import {Bomber, BonusListener, OwnershipsGetter, TileClicker, UpdatesListener} from "../../backends/backend.ts";
import BombNews from "../components/BombNews.tsx";
import {ChatBackend} from "../../backends/chat.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu from "../Menu.tsx";
import BonusAward from "../components/BonusAward.tsx";
import ClickBudgetMeter from "../components/ClickBudgetMeter.tsx";
import RateLimitModal from "../components/RateLimitModal.tsx";
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
import "./Viewer.css"

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudgetSource?: ClickBudgetSource
    bonusListener?: BonusListener
    bomber?: Bomber
    chatBackend?: ChatBackend
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const sound = useSound()

    const {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        rateLimited,
        dismissRateLimited,
        vpnBlocked,
        dismissVPNBlocked,
        sessionUnavailable,
        dismissSessionUnavailable,
        award,
        dismissAward,
        bonus,
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
        />}

        {status.state === 'ready' && <CameraButton busy={taking} onClick={take}/>}

        {shot && <SharePreview shot={shot}
                               stats={shareStats(leaderboard, countryState)}
                               onClose={discard}/>}

        {status.state === 'ready' && <ClickBudgetMeter budget={clickBudget} bonus={bonus} countryName={countryState.name}/>}

        {status.state === 'ready' && <ChatPanel
            backend={props.chatBackend}
            country={countryState}
            playSound={sound.play}
        />}

        {award && <BonusAward reward={award} onDone={dismissAward}/>}

        {lastBomb && <BombNews key={lastBomb.id} drop={lastBomb.drop} onDone={dismissBomb}/>}

        {rateLimited && <RateLimitModal onClose={dismissRateLimited}/>}

        {vpnBlocked && <VPNBlockedModal onClose={dismissVPNBlocked}/>}

        {sessionUnavailable && <SessionUnavailableModal onClose={dismissSessionUnavailable}/>}
    </>
}

function StatusCard({status}: {status: GlobeStatus}) {
    return <div className="viewer-status">
        <div className="viewer-status-card">
            {status.state === 'failed'
                ? <>
                    <h3>The globe could not be loaded</h3>
                    <p>{status.message}</p>
                </>
                : <>
                    <div className="viewer-status-spinner"/>
                    <h3>Loading the planet…</h3>
                </>}
        </div>
    </div>
}
