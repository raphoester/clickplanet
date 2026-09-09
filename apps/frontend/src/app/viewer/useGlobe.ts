import {useEffect, useRef, useState} from 'react';
import {createGlobe, Globe} from './globe.ts';
import {Country} from '../../domain/countries.ts';
import {LeaderboardEntry} from '../../domain/leaderboard.ts';
import {OwnershipsGetter, TileClicker, UpdatesListener} from '../../backends/backend.ts';

export type GlobeStatus =
    | {state: 'loading'}
    | {state: 'ready'}
    | {state: 'failed', message: string}

export type UseGlobeOptions = {
    container: React.RefObject<HTMLElement>
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    country: Country
}

/**
 * Owns one running globe for as long as the component is mounted, and exposes
 * what React needs to render around it.
 *
 * The effect deliberately depends on the three backends and nothing else. It
 * used to depend on the whole props object, which is a fresh reference on every
 * render, so any re-render of a parent would have torn down the WebGL context
 * and rebuilt the entire scene. The selected country is pushed in through a
 * separate effect for the same reason: it changes often, and rebuilding the
 * globe for it would be absurd.
 */
export function useGlobe(options: UseGlobeOptions) {
    const {container, tileClicker, ownershipsGetter, updatesListener, country} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])
    const [tilesCount, setTilesCount] = useState(0)

    /**
     * A flag rather than a count: the globe reports every refused click, and a
     * burst of them is one thing to tell the player, once.
     */
    const [rateLimited, setRateLimited] = useState(false)

    const globeRef = useRef<Globe | null>(null)

    /** Read once, when the globe is built; later changes go through setCountry. */
    const initialCountry = useRef(country)

    useEffect(() => {
        const element = container.current
        if (!element) return

        const abortController = new AbortController()
        let cancelled = false

        setStatus({state: 'loading'})

        createGlobe({
            tileClicker,
            ownershipsGetter,
            updatesListener,
            container: element,
            country: initialCountry.current,
            onLeaderboardChange: setLeaderboard,
            onRateLimited: () => setRateLimited(true),
            signal: abortController.signal,
        }).then((globe) => {
            if (cancelled) {
                globe.dispose()
                return
            }

            globeRef.current = globe
            setTilesCount(globe.tilesCount)
            setStatus({state: 'ready'})
        }).catch((error) => {
            if (cancelled) return
            console.error("Failed to initialize the globe", error)
            setStatus({state: 'failed', message: messageOf(error)})
        })

        return () => {
            cancelled = true
            abortController.abort()
            globeRef.current?.dispose()
            globeRef.current = null
        }
    }, [container, tileClicker, ownershipsGetter, updatesListener])

    useEffect(() => {
        initialCountry.current = country
        globeRef.current?.setCountry(country)
    }, [country])

    return {
        status,
        leaderboard,
        tilesCount,
        rateLimited,
        dismissRateLimited: () => setRateLimited(false),
    }
}

function messageOf(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}
