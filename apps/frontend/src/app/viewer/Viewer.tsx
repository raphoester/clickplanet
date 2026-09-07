import {useRef} from 'react';
import {OwnershipsGetter, TileClicker, UpdatesListener} from "../../backends/backend.ts";
import Settings from "../Settings.tsx";
import Leaderboard from "../Leaderboard.tsx";
import About from "../About.tsx";
import DiscordButton from '../components/DiscordButton.tsx';
import {useCountryStorage} from './useCountryStorage.ts';
import {GlobeStatus, useGlobe} from './useGlobe.ts';
import "./Viewer.css"

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
}

export default function Viewer(props: ViewerProps) {
    const container = useRef<HTMLDivElement>(null)
    const {countryState, handleSetCountry} = useCountryStorage()

    const {status, leaderboard, tilesCount} = useGlobe({
        container,
        tileClicker: props.tileClicker,
        ownershipsGetter: props.ownershipsGetter,
        updatesListener: props.updatesListener,
        country: countryState,
    })

    return <>
        <div ref={container} className="viewer-canvas"/>

        {status.state !== 'ready' && <StatusCard status={status}/>}

        {status.state === 'ready' && <div className="menu">
            <Leaderboard data={leaderboard} tilesCount={tilesCount}/>
            <div className="menu-actions">
                <Settings setCountry={handleSetCountry} country={countryState}/>
                <About/>
                <DiscordButton/>
            </div>
        </div>}
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
