import {useRef} from 'react';
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
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()
    const clickBudget = useClickBudget(props.clickBudgetSource, countryState.code)
    const sound = useSound()
    // The chat posts under the username, so it follows the account the menu shows.
    const account = useAccount(props.account)

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
        />}

        {status.state === 'ready' && <AnthemBar anthem={anthem}
                                                settings={sound.settings}
                                                onChange={sound.setSettings}/>}

        {status.state === 'ready' && <CameraButton busy={taking} onClick={take}/>}

        {shot && <SharePreview shot={shot}
                               stats={shareStats(leaderboard, countryState)}
                               onClose={discard}/>}

        {status.state === 'ready' && <ClickBudgetMeter budget={clickBudget} bonus={bonus} countryName={countryState.name} refusals={refusals}/>}

        {status.state === 'ready' && <ChatPanel
            backend={props.chatBackend}
            country={countryState}
            playSound={sound.play}
            username={account.kind === 'ready' ? account.username : undefined}
        />}

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
