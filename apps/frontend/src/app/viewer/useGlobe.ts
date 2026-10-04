import {useCallback, useEffect, useRef, useState} from 'react';
import {createGlobe, Globe} from './globe.ts';
import {CapturedFrame} from './capture.ts';
import {Country} from '../../domain/countries.ts';
import {OwnershipsGetter, TileClicker, UpdatesListener} from '../../backends/backend.ts';
import {useLeaderboardFeed} from './useLeaderboardFeed.ts';
import {ALL_OFF, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from '../../domain/bonus.ts';
import {BombDrop, Bomber, BonusCatch, BonusListener} from '../../backends/backend.ts';
import {PlaySound} from '../sound/soundPlayer.ts';
import {ClearNotes} from '../../domain/clearNotes.ts';
import {AcceptedClick} from './acceptedClicks.ts';

export type GlobeStatus =
    | {state: 'loading', territories?: number}
    | {state: 'ready'}
    | {state: 'failed', message: string}

export type UseGlobeOptions = {
    container: React.RefObject<HTMLElement>
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    bonusListener?: BonusListener
    bomber?: Bomber
    // Must keep its identity: a new one rebuilds the globe.
    playSound?: PlaySound
    // Must keep its identity: a new one rebuilds the globe.
    onClickAccepted?: (click: AcceptedClick) => void
    country: Country
    clickHue: number | undefined
}

export function useGlobe(options: UseGlobeOptions) {
    const {container, tileClicker, ownershipsGetter, updatesListener, bonusListener, bomber, playSound, onClickAccepted, country, clickHue} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [tilesCount, setTilesCount] = useState(0)

    const {leaderboard, tileDeltas, recordLeaderboard, publishLeaderboard} = useLeaderboardFeed()

    const [refusals, setRefusals] = useState(0)

    const [vpnBlocked, setVPNBlocked] = useState(false)

    const [sessionUnavailable, setSessionUnavailable] = useState(false)

    const [award, setAward] = useState<BonusReward | undefined>()
    const [charges, setCharges] = useState<Charges>(NO_CHARGES)
    const [rules, setRules] = useState<BonusRules | undefined>()
    const [bombArmed, setBombArmed] = useState(false)
    const [switches, setSwitches] = useState<Switches>(ALL_OFF)

    const [lastCatch, setLastCatch] = useState<BonusCatch | undefined>()

    const recordCatch = useCallback((taken: BonusCatch) => setLastCatch(taken), [])

    const [lastBomb, setLastBomb] = useState<{drop: BombDrop, land: string | undefined, id: number} | undefined>()
    const recordBomb = useCallback((drop: BombDrop, land: string | undefined) => {
        setLastBomb((previous) => ({drop, land, id: (previous?.id ?? 0) + 1}))
    }, [])

    const takeBonus = useCallback((reward: BonusReward) => setAward(reward), [])

    const [clearNotes] = useState(() => new ClearNotes(localStore()))
    const [lastClear, setLastClear] = useState<{ground: string, id: number} | undefined>()
    const recordClear = useCallback((ground: string) => {
        if (!clearNotes.due) return
        clearNotes.record()
        setLastClear((previous) => ({ground, id: (previous?.id ?? 0) + 1}))
    }, [clearNotes])

    const globeRef = useRef<Globe | null>(null)

    const initialCountry = useRef(country)
    const latestClickHue = useRef(clickHue)

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
            onLoadProgress: (territories) => {
                if (!cancelled) setStatus({state: 'loading', territories})
            },
            onRateLimited: () => setRefusals(n => n + 1),
            onVPNBlocked: () => setVPNBlocked(true),
            onSessionUnavailable: () => setSessionUnavailable(true),
            onBonusWon: takeBonus,
            onBonusTaken: recordCatch,
            onCharges: setCharges,
            onRules: setRules,
            onSwitchesChange: setSwitches,
            bonusListener,
            bomber,
            onBombDropped: recordBomb,
            onArmedChange: setBombArmed,
            onNativeCleared: recordClear,
            onClickAccepted,
            playSound,
            signal: abortController.signal,
        }).then((globe) => {
            if (cancelled) {
                globe.dispose()
                return
            }

            globeRef.current = globe
            globe.setClickHue(latestClickHue.current)
            if (import.meta.env.DEV) Object.assign(window, {clickplanetGlobe: globe})
            setTilesCount(globe.tilesCount)
            publishLeaderboard()
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
    }, [container, tileClicker, ownershipsGetter, updatesListener, bonusListener, bomber, playSound, onClickAccepted, recordLeaderboard, publishLeaderboard, takeBonus, recordCatch, recordBomb, recordClear])

    useEffect(() => {
        initialCountry.current = country
        globeRef.current?.setCountry(country)
    }, [country])

    useEffect(() => {
        latestClickHue.current = clickHue
        globeRef.current?.setClickHue(clickHue)
    }, [clickHue])

    const capture = useCallback((): Promise<CapturedFrame> => {
        const globe = globeRef.current
        if (!globe) return Promise.reject(new Error("the globe is not running yet"))
        return globe.capture()
    }, [])

    const dismissAward = useCallback(() => setAward(undefined), [])
    const toggleBomb = useCallback(() => globeRef.current?.setArmed(!bombArmed), [bombArmed])
    const toggleSwitch = useCallback((name: keyof Switches) => globeRef.current?.setSwitch(name, !switches[name]), [switches])
    const dismissBomb = useCallback(() => setLastBomb(undefined), [])
    const dismissClear = useCallback(() => setLastClear(undefined), [])

    return {
        status,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        refusals,
        vpnBlocked,
        dismissVPNBlocked: () => setVPNBlocked(false),
        sessionUnavailable,
        dismissSessionUnavailable: () => setSessionUnavailable(false),
        award,
        dismissAward,
        charges,
        rules,
        bombArmed,
        toggleBomb,
        switches,
        toggleSwitch,
        lastCatch,
        lastBomb,
        dismissBomb,
        lastClear,
        dismissClear,
    }
}

function localStore(): Storage | undefined {
    try {
        return window.localStorage
    } catch {
        return undefined
    }
}

function messageOf(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}
