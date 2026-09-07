import { useEffect, useRef, useState } from 'react';
import { effect, EffectHandle } from "./effect.ts";
import { OwnershipsGetter, TileClicker, UpdatesListener } from "../../backends/backend.ts";
import Settings from "../Settings.tsx";
import { Country } from "../../domain/countries.ts";
import { LeaderboardEntry } from "../../domain/leaderboard.ts";
import Leaderboard from "../Leaderboard.tsx";
import About from "../About.tsx";
import "./Viewer.css"
import { useCountryStorage } from './useCountryStorage.ts';
import DiscordButton from '../components/DiscordButton.tsx';

export type ViewerProps = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
}

export default function Viewer(props: ViewerProps) {

    const { countryState, handleSetCountry } = useCountryStorage()

    const setCountryRef = useRef<(country: Country) => void>();
    const tilesCountRef = useRef(0)

    const [leaderboardData, setLeaderboardData] = useState<LeaderboardEntry[]>([])
    const [isReady, setIsReady] = useState(false);
    const [loadError, setLoadError] = useState<string | null>(null);

    useEffect(() => {
        const eventTarget = document.getElementById("three-container")!

        // The globe now waits on a ~5 MB coordinates download, so init is async
        // while cleanup stays synchronous. Under <StrictMode> React tears the
        // first run down before it resolves, hence the cancelled flag: it aborts
        // the fetch and, if the run had already resolved, disposes it so the
        // discarded mount leaves no WebGL context behind.
        const abortController = new AbortController()
        let cancelled = false
        let handle: EffectHandle | null = null

        setIsReady(false)
        setLoadError(null)

        effect(
            props.tileClicker,
            props.ownershipsGetter,
            props.updatesListener,
            (data) => setLeaderboardData(data),
            eventTarget,
            countryState,
            abortController.signal,
        ).then((result) => {
            if (cancelled) {
                result.cleanup()
                return
            }

            handle = result
            tilesCountRef.current = result.tilesCount
            setCountryRef.current = result.updateCountry
            setIsReady(true)
        }).catch((error) => {
            if (cancelled) return
            console.error("Failed to initialize the globe", error)
            setLoadError(error instanceof Error ? error.message : String(error))
        })

        return () => {
            cancelled = true
            abortController.abort()
            handle?.cleanup()
            handle = null
        }
    }, [props]);

    const setCountry = (country: Country) => {
        handleSetCountry(country)
        setCountryRef.current?.(country)
    }

    return <>
        <div>
            <div id="three-container" style={{ width: '100vw', height: '100vh' }} />
        </div>
        {!isReady && <div className="viewer-status">
            <div className="viewer-status-card">
                {loadError
                    ? <>
                        <h3>The globe could not be loaded</h3>
                        <p>{loadError}</p>
                    </>
                    : <>
                        <div className="viewer-status-spinner"/>
                        <h3>Loading the planet…</h3>
                    </>}
            </div>
        </div>}
        <div className="menu">
            {isReady && <>
                <Leaderboard
                    data={leaderboardData}
                    tilesCount={tilesCountRef.current}
                />
                <div className="menu-actions">
                    <Settings
                        setCountry={setCountry}
                        country={countryState}
                    />
                    <About/>
                    <DiscordButton />
                </div>
            </>}
        </div>
    </>
};
