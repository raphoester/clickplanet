import {useCallback, useEffect, useRef, useState} from 'react';
import {createGlobe, Globe} from './globe.ts';
import {CapturedFrame} from './capture.ts';
import {Country} from '../../domain/countries.ts';
import {MapView, Rendering} from '../../domain/displaySettings.ts';
import {OwnershipsGetter, TileClicker, UpdatesListener} from '../../backends/backend.ts';
import {useLeaderboardFeed} from './useLeaderboardFeed.ts';
import {ALL_OFF, BonusNotice, BonusNoticeEvent, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from '../../domain/bonus.ts';
import {BombDrop, Bomber, BonusCatch, BonusListener, Shielder} from '../../backends/backend.ts';
import {PlaySound} from '../sound/soundPlayer.ts';
import {AcceptedClick} from './acceptedClicks.ts';
import {CloseToFortify, Fortified} from '../../domain/fortify.ts';
import {ClickBudgetSource} from '../../backends/clickBudget.ts';

export type GlobeStatus =
    | {state: 'loading', territories?: number}
    | {state: 'ready'}
    | {state: 'failed', message: string}

export type UseGlobeOptions = {
    container: React.RefObject<HTMLElement>
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudget?: ClickBudgetSource
    bonusListener?: BonusListener
    bomber?: Bomber
    shielder?: Shielder
    // Must keep its identity: a new one rebuilds the globe.
    playSound?: PlaySound
    // Must keep its identity: a new one rebuilds the globe.
    onClickAccepted?: (click: AcceptedClick) => void
    country: Country
    clickHue: number | undefined
    mapView: MapView
    // A change rebuilds the globe: antialiasing is fixed when the context is made.
    rendering: Rendering
}

export function useGlobe(options: UseGlobeOptions) {
    const {container, tileClicker, ownershipsGetter, updatesListener, clickBudget, bonusListener, bomber, shielder, playSound, onClickAccepted, country, clickHue, mapView, rendering} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [tilesCount, setTilesCount] = useState(0)

    const {leaderboard, tileDeltas, recordLeaderboard, publishLeaderboard} = useLeaderboardFeed()

    const [refusals, setRefusals] = useState(0)

    const [notice, setNotice] = useState<BonusNoticeEvent | undefined>()
    const notify = useCallback((what: BonusNotice) => setNotice((was) => ({notice: what, seq: (was?.seq ?? 0) + 1})), [])

    const [vpnBlocked, setVPNBlocked] = useState(false)

    const [sessionUnavailable, setSessionUnavailable] = useState(false)

    const [award, setAward] = useState<BonusReward | undefined>()
    const [charges, setCharges] = useState<Charges>(NO_CHARGES)
    const [rules, setRules] = useState<BonusRules | undefined>()
    const [bombArmed, setBombArmed] = useState(false)
    const [switches, setSwitches] = useState<Switches>(ALL_OFF)

    const [lastCatch, setLastCatch] = useState<BonusCatch | undefined>()

    const recordCatch = useCallback((taken: BonusCatch) => setLastCatch(taken), [])

    const [lastBomb, setLastBomb] = useState<{drop: BombDrop, land: string | undefined, id: number, at: number} | undefined>()
    const recordBomb = useCallback((drop: BombDrop, land: string | undefined) => {
        setLastBomb((previous) => ({drop, land, id: (previous?.id ?? 0) + 1, at: performance.now()}))
    }, [])

    const [lastFortified, setLastFortified] = useState<{fortified: Fortified, id: number, at: number} | undefined>()
    const recordFortified = useCallback((fortified: Fortified) => {
        if (!fortified.news) return
        setLastFortified((previous) => ({fortified, id: (previous?.id ?? 0) + 1, at: performance.now()}))
    }, [])

    const [lastClose, setLastClose] = useState<{close: CloseToFortify, id: number, at: number} | undefined>()
    const recordClose = useCallback((close: CloseToFortify) => {
        setLastClose((previous) => ({close, id: (previous?.id ?? 0) + 1, at: performance.now()}))
    }, [])

    const takeBonus = useCallback((reward: BonusReward) => setAward(reward), [])

    const globeRef = useRef<Globe | null>(null)

    const initialCountry = useRef(country)
    const latestClickHue = useRef(clickHue)
    const latestMapView = useRef(mapView)

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
            clickBudget,
            container: element,
            country: initialCountry.current,
            mapView: latestMapView.current,
            rendering,
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
            shielder,
            onNotice: notify,
            onClickAccepted,
            onFortified: recordFortified,
            onCloseToFortify: recordClose,
            playSound,
            signal: abortController.signal,
        }).then((globe) => {
            if (cancelled) {
                globe.dispose()
                return
            }

            globeRef.current = globe
            globe.setClickHue(latestClickHue.current)
            globe.setMapView(latestMapView.current)
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
    }, [container, tileClicker, ownershipsGetter, updatesListener, clickBudget, bonusListener, bomber, shielder, playSound, onClickAccepted, rendering, recordLeaderboard, publishLeaderboard, takeBonus, recordCatch, recordBomb, recordFortified, recordClose, notify])

    useEffect(() => {
        initialCountry.current = country
        globeRef.current?.setCountry(country)
    }, [country])

    useEffect(() => {
        latestClickHue.current = clickHue
        globeRef.current?.setClickHue(clickHue)
    }, [clickHue])

    useEffect(() => {
        latestMapView.current = mapView
        globeRef.current?.setMapView(mapView)
    }, [mapView])

    const capture = useCallback((): Promise<CapturedFrame> => {
        const globe = globeRef.current
        if (!globe) return Promise.reject(new Error("the globe is not running yet"))
        return globe.capture()
    }, [])

    const dismissAward = useCallback(() => setAward(undefined), [])
    const toggleBomb = useCallback(() => globeRef.current?.setArmed(!bombArmed), [bombArmed])
    const toggleSwitch = useCallback((name: keyof Switches) => globeRef.current?.setSwitch(name, !switches[name]), [switches])
    const dismissBomb = useCallback(() => setLastBomb(undefined), [])
    const dismissFortified = useCallback(() => setLastFortified(undefined), [])
    const dismissClose = useCallback(() => setLastClose(undefined), [])

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
        notice,
        lastCatch,
        lastBomb,
        dismissBomb,
        lastFortified,
        dismissFortified,
        lastClose,
        dismissClose,
    }
}

function messageOf(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
}
