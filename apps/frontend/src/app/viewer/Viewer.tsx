import {useRef} from 'react';
import {OwnershipsGetter, TileClicker, UpdatesListener} from "../../backends/backend.ts";
import {ChatBackend} from "../../backends/chat.ts";
import ChatPanel from "../chat/ChatPanel.tsx";
import Menu from "../Menu.tsx";
import RateLimitModal from "../components/RateLimitModal.tsx";
import SessionUnavailableModal from "../components/SessionUnavailableModal.tsx";
import VPNBlockedModal from "../components/VPNBlockedModal.tsx";
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
import "./Viewer.css"

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    chatBackend?: ChatBackend
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()

    const {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        rateLimited,
        dismissRateLimited,
        vpnBlocked,
        dismissVPNBlocked,
        sessionUnavailable,
        dismissSessionUnavailable,
    } = useGlobe({
        container,
        tileClicker: props.tileClicker,
        ownershipsGetter: props.ownershipsGetter,
        updatesListener: props.updatesListener,
        country: countryState,
    })

    return <>
        <div ref={container} className="viewer-canvas"/>

        {status.state !== 'ready' && <StatusCard status={status}/>}

        {status.state === 'ready' && <Menu
            country={countryState}
            setCountry={handleSetCountry}
            leaderboard={leaderboard}
            tileDeltas={tileDeltas}
            tilesCount={tilesCount}
        />}

        {status.state === 'ready' && <ChatPanel
            backend={props.chatBackend}
            country={countryState}
        />}

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
