import {useCallback, useEffect, useRef, useState} from 'react';
import {createGlobe, Globe} from './globe.ts';
import {Halo} from './atmosphere.ts';
import {CapturedFrame} from './capture.ts';
import {GraphicsReport, NO_GRAPHICS} from './graphicsReport.ts';
import {Country} from '../../domain/countries.ts';
import {MapView, Rendering} from '../../domain/displaySettings.ts';
import {OwnershipsGetter, TileClicker, UpdatesListener} from '../../backends/backend.ts';
import {useLeaderboardFeed} from './useLeaderboardFeed.ts';
import {Switches} from '../../domain/bonus.ts';
import {Bomber, BonusListener, Shielder} from '../../backends/backend.ts';
import {PlaySound} from '../sound/soundPlayer.ts';
import {AcceptedClick} from '../clicks/acceptedClicks.ts';
import {ClickBudgetSource} from '../../backends/clickBudget.ts';
import {useRefusedClicks} from '../clicks/refusedClicksStore.ts';
import {useBonus} from '../bonus/bonusStore.ts';
import {useNews} from '../news/newsStore.ts';
import {useStanding} from '../knockout/standingStore.ts';
import {Peek} from '../../domain/tilePeek.ts';
import {Territory, TerritoryPeek} from '../../domain/territory.ts';
import {Shot} from '../../domain/clip/camera.ts';
import {MissionTarget} from '../../domain/missionTargets.ts';

export type GlobeStatus =
    | {state: 'loading', territories?: number}
    | {state: 'ready'}
    | {state: 'failed'}

export type UseGlobeOptions = {
    container: React.RefObject<HTMLElement>
    // Must keep its identity: a new one rebuilds the globe.
    linkedShot?: Shot
    // A change rebuilds the globe: a first visit is guided to its front from the load.
    showFront?: boolean
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
    // The players online, when known: the ghosts of a first visit raid only while few play.
    crowd?: number
    // A battle runs: its rules are the game, and no ghost raids.
    battle: boolean
    clickHue: number | undefined
    mapView: MapView
    halo: Halo
    // Each change is a scan of every tile: keep its identity while the targets are the same.
    missions: readonly MissionTarget[]
    // A change rebuilds the globe: antialiasing is fixed when the context is made.
    rendering: Rendering
}

export function useGlobe(options: UseGlobeOptions) {
    const {container, linkedShot, showFront, tileClicker, ownershipsGetter, updatesListener, clickBudget, bonusListener, bomber, shielder, playSound, onClickAccepted, country, crowd, battle, clickHue, mapView, halo, missions, rendering} = options

    const [status, setStatus] = useState<GlobeStatus>({state: 'loading'})
    const [tilesCount, setTilesCount] = useState(0)
    // The share of the tiles read, until they all are.
    const [tilesLoading, setTilesLoading] = useState<number | undefined>(0)
    const [orbit, setOrbit] = useState(true)

    const {leaderboard, tileDeltas, recordLeaderboard, publishLeaderboard} = useLeaderboardFeed()

    const [pointing, setPointing] = useState<{peek: Peek, id: number} | undefined>()
    const peekCount = useRef(0)
    const recordPeek = useCallback((peek: Peek) => setPointing({peek, id: ++peekCount.current}), [])
    const endPeek = useCallback(() => setPointing(undefined), [])

    const [pointingTerritory, setPointingTerritory] = useState<{peek: TerritoryPeek, id: number} | undefined>()
    const recordTerritoryPeek = useCallback((peek: TerritoryPeek) => setPointingTerritory({peek, id: ++peekCount.current}), [])
    const endTerritoryPeek = useCallback(() => setPointingTerritory(undefined), [])

    const globeRef = useRef<Globe | null>(null)

    const initialCountry = useRef(country)
    const latestCrowd = useRef(crowd)
    const latestBattle = useRef(battle)
    const latestClickHue = useRef(clickHue)
    const latestMapView = useRef(mapView)
    const latestHalo = useRef(halo)
    const latestMissions = useRef(missions)

    useEffect(() => {
        const element = container.current
        if (!element) return

        const abortController = new AbortController()
        let cancelled = false

        setStatus({state: 'loading'})
        setTilesLoading(0)

        const refused = useRefusedClicks.getState()
        const bonus = useBonus.getState()
        const news = useNews.getState()
        const standing = useStanding.getState()

        createGlobe({
            tileClicker,
            ownershipsGetter,
            updatesListener,
            clickBudget,
            container: element,
            linkedShot,
            showFront,
            country: initialCountry.current,
            mapView: latestMapView.current,
            rendering,
            onLeaderboardChange: recordLeaderboard,
            onLoadProgress: (share) => {
                if (!cancelled) setTilesLoading(share < 1 ? share : undefined)
            },
            onOrbitChange: (inOrbit) => {
                if (!cancelled) setOrbit(inOrbit)
            },
            onRateLimited: refused.refuse,
            onFrozen: refused.refuse,
            onVPNBlocked: refused.blockVPN,
            onBanned: refused.ban,
            onSessionUnavailable: refused.failSession,
            onAllegianceLocked: refused.lockOut,
            onOutOfReach: () => {
                refused.refuseOutOfReach()
                standing.ask()
            },
            onStanding: standing.follow,
            onBonusWon: bonus.win,
            onCharges: bonus.setCharges,
            onRules: bonus.setRules,
            onSwitchesChange: bonus.setSwitches,
            bonusListener,
            bomber,
            onBombDropped: news.tellBomb,
            onArmedChange: bonus.setBombArmed,
            shielder,
            onNotice: bonus.notify,
            onClickAccepted,
            onFortified: news.tellFortified,
            onPeek: recordPeek,
            onPeekEnd: endPeek,
            onTerritoryPeek: recordTerritoryPeek,
            onTerritoryPeekEnd: endTerritoryPeek,
            playSound,
            signal: abortController.signal,
        }).then((globe) => {
            if (cancelled) {
                globe.dispose()
                return
            }

            globeRef.current = globe
            globe.setCrowd(latestCrowd.current)
            globe.setBattle(latestBattle.current)
            globe.setClickHue(latestClickHue.current)
            globe.setMapView(latestMapView.current)
            globe.setHalo(latestHalo.current)
            globe.setMissions(latestMissions.current)
            if (import.meta.env.DEV) Object.assign(window, {clickplanetGlobe: globe})
            setTilesCount(globe.tilesCount)
            publishLeaderboard()
            setStatus({state: 'ready'})
        }).catch((error) => {
            if (cancelled) return
            console.error("Failed to initialize the globe", error)
            setStatus({state: 'failed'})
        })

        return () => {
            cancelled = true
            abortController.abort()
            globeRef.current?.dispose()
            globeRef.current = null
        }
    }, [container, linkedShot, showFront, tileClicker, ownershipsGetter, updatesListener, clickBudget, bonusListener, bomber, shielder, playSound, onClickAccepted, rendering, recordLeaderboard, publishLeaderboard, recordPeek, endPeek, recordTerritoryPeek, endTerritoryPeek])

    useEffect(() => () => {
        useRefusedClicks.setState(useRefusedClicks.getInitialState(), true)
        useBonus.setState(useBonus.getInitialState(), true)
        useNews.setState(useNews.getInitialState(), true)
        useStanding.setState(useStanding.getInitialState(), true)
    }, [])

    useEffect(() => {
        initialCountry.current = country
        globeRef.current?.setCountry(country)
    }, [country])

    useEffect(() => {
        latestCrowd.current = crowd
        globeRef.current?.setCrowd(crowd)
    }, [crowd])

    useEffect(() => {
        latestBattle.current = battle
        globeRef.current?.setBattle(battle)
    }, [battle])

    useEffect(() => {
        latestClickHue.current = clickHue
        globeRef.current?.setClickHue(clickHue)
    }, [clickHue])

    useEffect(() => {
        latestMapView.current = mapView
        globeRef.current?.setMapView(mapView)
    }, [mapView])

    useEffect(() => {
        latestHalo.current = halo
        globeRef.current?.setHalo(halo)
    }, [halo])

    useEffect(() => {
        latestMissions.current = missions
        globeRef.current?.setMissions(missions)
    }, [missions])

    const capture = useCallback((): Promise<CapturedFrame> => {
        const globe = globeRef.current
        if (!globe) return Promise.reject(new Error("the globe is not running yet"))
        return globe.capture()
    }, [])

    const graphics = useCallback((): GraphicsReport => globeRef.current?.graphics() ?? NO_GRAPHICS, [])

    const territoryOf = useCallback((landmass: number): Territory | undefined => globeRef.current?.territoryOf(landmass), [])
    const showTerritory = useCallback((landmass: number | undefined) => globeRef.current?.showTerritory(landmass), [])
    const flyToLastLoss = useCallback(() => globeRef.current?.flyToLastLoss(), [])
    const flyToMission = useCallback((target: MissionTarget) => globeRef.current?.flyToMission(target), [])

    const toggleBomb = useCallback(() => globeRef.current?.setArmed(!useBonus.getState().bombArmed), [])
    const toggleSwitch = useCallback((name: keyof Switches) => globeRef.current?.setSwitch(name, !useBonus.getState().switches[name]), [])

    return {
        status,
        tilesLoading,
        orbit,
        leaderboard,
        tileDeltas,
        tilesCount,
        capture,
        graphics,
        toggleBomb,
        toggleSwitch,
        pointing,
        pointingTerritory,
        territoryOf,
        showTerritory,
        flyToLastLoss,
        flyToMission,
    }
}
