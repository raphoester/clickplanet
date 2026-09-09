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

export function useGlobe(options: UseGlobeOptions) {
    const {container, tileClicker, ownershipsGetter, updatesListener, country} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])
    const [tilesCount, setTilesCount] = useState(0)

    const [rateLimited, setRateLimited] = useState(false)

    const [vpnBlocked, setVPNBlocked] = useState(false)

    const [sessionUnavailable, setSessionUnavailable] = useState(false)

    const globeRef = useRef<Globe | null>(null)

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
            onVPNBlocked: () => setVPNBlocked(true),
            onSessionUnavailable: () => setSessionUnavailable(true),
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
        vpnBlocked,
        dismissVPNBlocked: () => setVPNBlocked(false),
        sessionUnavailable,
        dismissSessionUnavailable: () => setSessionUnavailable(false),
    }
}

function messageOf(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}
