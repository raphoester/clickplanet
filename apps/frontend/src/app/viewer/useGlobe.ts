import {useCallback, useEffect, useRef, useState} from 'react';
import {createGlobe, Globe} from './globe.ts';
import {CapturedFrame} from './capture.ts';
import {Country} from '../../domain/countries.ts';
import {OwnershipsGetter, TileClicker, UpdatesListener} from '../../backends/backend.ts';
import {useLeaderboardFeed} from './useLeaderboardFeed.ts';
import {ActiveBonus, BonusReward} from '../../domain/bonus.ts';
import {BonusCatch, BonusListener} from '../../backends/backend.ts';
import {now} from '../../backends/clickBudget.ts';

export type GlobeStatus =
    | {state: 'loading'}
    | {state: 'ready'}
    | {state: 'failed', message: string}

export type UseGlobeOptions = {
    container: React.RefObject<HTMLElement>
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    /** Absent for a backend with no bonus feed, which draws no boxes at all. */
    bonusListener?: BonusListener
    country: Country
}

export function useGlobe(options: UseGlobeOptions) {
    const {container, tileClicker, ownershipsGetter, updatesListener, bonusListener, country} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [tilesCount, setTilesCount] = useState(0)

    const {leaderboard, tileDeltas, recordLeaderboard} = useLeaderboardFeed()

    const [rateLimited, setRateLimited] = useState(false)

    const [vpnBlocked, setVPNBlocked] = useState(false)

    const [sessionUnavailable, setSessionUnavailable] = useState(false)

    // Two pieces of state, because they have two lifetimes. `award` is the
    // two-second announcement; `bonus` is the reward itself, which outlives it
    // by a minute and is what the meter reads.
    const [award, setAward] = useState<BonusReward | undefined>()
    const [bonus, setBonus] = useState<ActiveBonus | undefined>()

    // Somebody caught one, anywhere on the planet. Held as the latest catch so
    // the board can say so; it is never what starts this client's own bonus,
    // which only the server's answer to its own claim does.
    const [lastCatch, setLastCatch] = useState<BonusCatch | undefined>()

    const recordCatch = useCallback((taken: BonusCatch) => setLastCatch(taken), [])

    const takeBonus = useCallback((reward: BonusReward) => {
        setAward(reward)
        // Stamped on the same monotonic clock as a budget reading, so the
        // countdown measures how long this machine has watched rather than
        // trusting a server timestamp from an unrelated clock.
        setBonus({reward, endsAt: now() + reward.seconds * 1000})
    }, [])

    // The bonus takes itself off, so nothing has to remember to. A second box
    // caught mid-bonus replaces the whole thing, and this effect re-runs with
    // the new deadline rather than leaving the old timer to cut it short.
    useEffect(() => {
        if (!bonus) return

        const timer = setTimeout(() => setBonus(undefined), Math.max(0, bonus.endsAt - now()))
        return () => clearTimeout(timer)
    }, [bonus])

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
            onLeaderboardChange: recordLeaderboard,
            onRateLimited: () => setRateLimited(true),
            onVPNBlocked: () => setVPNBlocked(true),
            onSessionUnavailable: () => setSessionUnavailable(true),
            onBonusWon: takeBonus,
            onBonusTaken: recordCatch,
            bonusListener,
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
    }, [container, tileClicker, ownershipsGetter, updatesListener, bonusListener, recordLeaderboard, takeBonus, recordCatch])

    useEffect(() => {
        initialCountry.current = country
        globeRef.current?.setCountry(country)
    }, [country])

    const capture = useCallback((): Promise<CapturedFrame> => {
        const globe = globeRef.current
        if (!globe) return Promise.reject(new Error("the globe is not running yet"))
        return globe.capture()
    }, [])

    const dismissAward = useCallback(() => setAward(undefined), [])

    return {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        rateLimited,
        dismissRateLimited: () => setRateLimited(false),
        vpnBlocked,
        dismissVPNBlocked: () => setVPNBlocked(false),
        sessionUnavailable,
        dismissSessionUnavailable: () => setSessionUnavailable(false),
        award,
        dismissAward,
        bonus,
        lastCatch,
    }
}

function messageOf(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}
