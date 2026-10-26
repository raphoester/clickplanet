import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, pixelRatio, setupScene} from "./scene.ts";
import {graphicsOf} from "./graphics.ts";
import {loadPointGeometryData} from "./points.ts";
import {GpuPicker} from "./gpuPicking.ts";
import {innerSphere} from "./sphere.ts";
import {CapturedFrame, readDrawingBuffer} from "./capture.ts";
import {GraphicsReport, graphicsReportOf} from "./graphicsReport.ts";
import {TileField} from "./tileField.ts";
import {BorderField, countryOfTile, loadBorders} from "./borderField.ts";
import {createBorderLines, loadBorderLines} from "./borderLines.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import {BORDERS_URL} from "./bordersAsset.ts";
import {BORDER_LINES_URL} from "./borderLinesAsset.ts";
import {displayPointSize, flagPaint, flownZoomsOf, inOrbit, pickPointSize} from "./pointSize.ts";
import {regions} from "./atlas.ts";
import {SlotRanking} from "./slotRanking.ts";
import {FlagSlots} from "./flagSlots.ts";
import {Country} from "../../domain/countries.ts";
import {Allegiance} from "../../domain/allegiance.ts";
import {MapView, Rendering} from "../../domain/displaySettings.ts";
import {
    AllegianceLockedError,
    BannedError,
    BombDrop,
    Bomber,
    GlobePoint,
    BonusCatch,
    Counts,
    MapVersion,
    BonusListener,
    BonusLostError,
    BonusOffer,
    ClaimedBonus,
    ShieldRefusedError,
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    Shielder,
    TileClicker,
    TileRange,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "../../backends/backend.ts";
import {SessionUnavailableError} from "../../backends/session.ts";
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts";
import {OwnerChange} from "../../domain/tileOwnership.ts";
import {MapTiles, TileChanges} from "../../domain/mapTiles.ts";
import {MapClock} from "../../domain/mapClock.ts";
import {Holdings, Recount} from "../../domain/holdings.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {layoutViewport} from "./viewport.ts";
import {createStarfield} from "./stars.ts";
import {MAX_ZOOM, MIN_ZOOM, RESTING_ZOOM} from "./zoom.ts";
import {createBonusBox} from "./bonusBox.ts";
import {createBonusPointer} from "./bonusPointer.ts";
import {createEnclosureEffects} from "./enclosureEffect.ts";
import {createClickEffects} from "./clickEffects.ts";
import {createClickGlints} from "./clickGlints.ts";
import {ALL_OFF, BonusNotice, BonusReward, BonusRules, Charges, NO_CHARGES, Switches, switched, switchesHeld} from "../../domain/bonus.ts";
import {canClick, ClickBudget, ClickBudgetSource, now as monotonicNow} from "../../backends/clickBudget.ts";
import {BlastUniforms, blastUniforms, createBlasts} from "./blasts.ts";
import {IMPACT_DELAY} from "../../domain/blast.ts";
import {HoldToDrop} from "../../domain/holdToDrop.ts";
import {ClickOrDrag} from "../../domain/clickOrDrag.ts";
import {Peek, Peeked, Point, TilePeek} from "../../domain/tilePeek.ts";
import {rankHoldings, Territory, TerritoryPeek} from "../../domain/territory.ts";
import {FLIGHT_SECONDS, flightAt, territoryShotOf} from "../../domain/flight.ts";
import {pointOf} from "../../domain/clip/geometry.ts";
import {OwnClicks} from "../../domain/ownClicks.ts";
import {ShieldChange, outcomeOf, placementOf} from "../../domain/shields.ts";
import {Landmasses} from "../../domain/landmasses.ts";
import {landmassName} from "../../domain/landmassNames.ts";
import {CloseToFortify, Fortified, isNews} from "../../domain/fortify.ts";
import {Entered, FortifyTargets} from "../../domain/fortifyTargets.ts";
import {createFortifyWaves, FortifyUniforms, fortifyUniforms} from "./fortifyWaves.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import {drawShieldMarks, shieldCellsOf} from "./shieldMarks.ts";
import {AcceptedClick} from "./acceptedClicks.ts";

type Uniforms = BlastUniforms & FortifyUniforms & {
    pointSize: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
    landmassData: THREE.IUniform
    landmassCount: THREE.IUniform
    pixelsPerRadian: THREE.IUniform<number>
    pixelRatio: THREE.IUniform<number>
    flagPaint: THREE.IUniform
    shieldMost: THREE.IUniform<number>
    shieldMarks: THREE.IUniform<THREE.Texture>
    shieldCells: THREE.IUniform<THREE.Vector2>
    highlight: THREE.IUniform<number>
    slotTexture: THREE.IUniform<THREE.Texture | null>
    slotFrames: THREE.IUniform<THREE.Vector4[]>
}

const SHAKE_SECONDS = 0.5

const HOLD_TO_DROP_SECONDS = 0.7

const HOLD_TOLERANCE_PX = 6

const CLICK_TOLERANCE = {mousePx: 6, touchPx: 12}

const PEEK_RULES = {restSeconds: 0.5, holdSeconds: 0.5, tolerancePx: CLICK_TOLERANCE.touchPx}

const DISTANT_BOMB_VOLUME = 0.45

const OWN_DROP_WINDOW_SECONDS = 5

const OWN_CLICK_WINDOW_SECONDS = 3

const ENCLOSURE_WAIT_MS = 1500

const SHIELD_MOST_UNTIL_READ = 10

const TILES_PER_BATCH = 10_000

// GetMap's and GetHoldings' max-age: a read any sooner can be the same old copy.
const READ_AGAIN_MS = 5_000

const READS_AT_MOST = 4

const IDLE_FRAME_MS = 16

// A pulse that only says "look here" is drawn at 20 frames a second, to spare the battery.
const AMBIENT_FRAME_MS = 50

const SPIN_TURNS_PER_MINUTE = 2

const MAX_SPIN_STEP_MS = 100

export function spinStep(sinceLastTick: number): number {
    return Math.min(Math.max(sinceLastTick, 0), MAX_SPIN_STEP_MS) / 1000
}

const INTERACTION_GRACE_MS = 1_000

export type Tick = {
    turned: boolean
    changed: boolean
    ambient: boolean
    at: number
    drawnAt: number
    interactingUntil: number
    sinceLastTick: number
}

export function drawsFrame({turned, changed, ambient, at, drawnAt, interactingUntil, sinceLastTick}: Tick): boolean {
    if (!turned && !changed && !ambient) return false
    if (changed || at <= interactingUntil) return true
    return at + sinceLastTick / 2 >= drawnAt + (turned ? IDLE_FRAME_MS : AMBIENT_FRAME_MS)
}

const CLAIM_MARGIN_MS = 2_000

const textureLoader = new THREE.TextureLoader();

export type Shot = {
    direction: GlobePoint
    zoom: number
}

export type Director = (seconds: number) => Shot

// Flies the camera now and then, and lets go as soon as the player takes the globe.
type Pilot = {
    shotAt(seconds: number): Shot | undefined
    cancel(): void
}

type Flight = {from: Shot, to: Shot, since: number, seconds: number}

export type GlobeOptions = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudget?: ClickBudgetSource
    container: HTMLElement
    country: Country
    mapView: MapView
    rendering: Rendering
    onLeaderboardChange: (entries: LeaderboardEntry[], live: boolean) => void
    // The share of the tiles read, after the globe is ready: a tile is clicked only once they all are.
    onLoadProgress: (share: number) => void
    onOrbitChange?: (orbit: boolean) => void
    onRateLimited: () => void
    onVPNBlocked: () => void
    onBanned: (until: number) => void
    onSessionUnavailable: () => void
    onAllegianceLocked: (allegiance: Allegiance) => void
    onBonusTaken: (taken: BonusCatch) => void
    onBonusWon: (reward: BonusReward) => void
    onCharges: (charges: Charges) => void
    onRules: (rules: BonusRules) => void
    onSwitchesChange: (switches: Switches) => void
    bonusListener?: BonusListener
    bomber?: Bomber
    onBombDropped: (drop: BombDrop, land: string | undefined) => void
    onArmedChange: (armed: boolean) => void
    shielder?: Shielder
    onNotice?: (notice: BonusNotice) => void
    onClickAccepted?: (click: AcceptedClick) => void
    onFortified?: (fortified: Fortified) => void
    onCloseToFortify?: (close: CloseToFortify) => void
    onPeek?: (peek: Peek) => void
    onPeekEnd?: () => void
    onTerritoryPeek?: (peek: TerritoryPeek) => void
    onTerritoryPeekEnd?: () => void
    playSound?: PlaySound
    director?: Director
    signal: AbortSignal
}

export type Globe = {
    readonly tilesCount: number
    // Every tile read once: a tile can be clicked from then on.
    readonly tilesRead: Promise<void>
    setCountry(country: Country): void
    setMapView(view: MapView): void
    takeReward(claimed: ClaimedBonus): void
    setArmed(armed: boolean): void
    setSwitch(name: keyof Switches, on: boolean): void
    setClickHue(hue: number | undefined): void
    territoryOf(landmass: number): Territory
    showTerritory(landmass: number | undefined): void
    capture(): Promise<CapturedFrame>
    graphics(): GraphicsReport
    dispose(): void
}

type CaptureRequest = {
    resolve: (frame: CapturedFrame) => void
    reject: (reason: Error) => void
}

export async function createGlobe(options: GlobeOptions): Promise<Globe> {
    const {
        tileClicker,
        ownershipsGetter,
        updatesListener,
        clickBudget,
        container: eventTarget,
        country: initialCountry,
        mapView: initialMapView,
        rendering,
        onLeaderboardChange: updateLeaderboard,
        onLoadProgress,
        onOrbitChange = () => {},
        onRateLimited,
        onVPNBlocked,
        onBanned,
        onSessionUnavailable,
        onAllegianceLocked,
        onBonusTaken,
        onBonusWon,
        onCharges,
        onRules,
        onSwitchesChange,
        bonusListener,
        bomber,
        onBombDropped,
        onArmedChange,
        shielder,
        onNotice = () => {},
        onClickAccepted = () => {},
        onFortified = () => {},
        onCloseToFortify = () => {},
        onPeek = () => {},
        onPeekEnd = () => {},
        onTerritoryPeek = () => {},
        onTerritoryPeekEnd = () => {},
        playSound = () => {},
        director,
        signal,
    } = options

    const [geometryData, borders, borderLines] = await Promise.all([
        loadPointGeometryData(signal),
        loadBorders(BORDERS_URL, signal),
        loadBorderLines(BORDER_LINES_URL, signal),
    ]);
    if (signal.aborted) throw new DOMException("globe load aborted", "AbortError");

    const lifetime = new AbortController();
    const listenerOptions = {signal: lifetime.signal};

    let budget: ClickBudget | undefined
    const stopWatchingBudget = clickBudget?.watchClickBudget((reading) => {
        budget = reading
    })

    let loaded = false

    const graphics = graphicsOf(window.location.search, rendering);
    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget, graphics);
    let mapView: MapView = initialMapView

    let dirty = true
    const invalidate = () => {
        dirty = true
    }

    const slots = new FlagSlots(invalidate)

    const uniforms: Uniforms = {
        pointSize: {value: displayPointSize(camera.zoom, layoutViewport().height, mapView) * renderer.getPixelRatio()},
        atlasTexture: {value: textureLoader.load(ATLAS_URL)},
        atlasTextureSize: {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)},
        landmassData: {value: null},
        landmassCount: {value: 1},
        pixelsPerRadian: {value: 1},
        pixelRatio: {value: renderer.getPixelRatio()},
        flagPaint: {value: flagPaint(camera.zoom, layoutViewport().height, mapView)},
        shieldMost: {value: SHIELD_MOST_UNTIL_READ},
        shieldMarks: {value: drawShieldMarks(SHIELD_MOST_UNTIL_READ, invalidate)},
        shieldCells: {value: shieldCellsOf(SHIELD_MOST_UNTIL_READ)},
        highlight: {value: 0},
        slotTexture: slots.texture,
        slotFrames: slots.frames,
        ...blastUniforms(prefersReducedMotion()),
        ...fortifyUniforms(),
    };

    const pickingUniforms = {pointSize: {value: pickPointSize(camera.zoom, layoutViewport().height) * renderer.getPixelRatio()}}

    const field = new TileField(uniforms, pickingUniforms, geometryData, graphics);

    const territories = new BorderField(borders)
    const landmasses = new Landmasses(borders.assignment, borders.codes.length)
    field.setLandmasses(borders.assignment)
    uniforms.landmassData.value = territories.landmassData
    uniforms.landmassCount.value = borders.codes.length

    const ranking = new SlotRanking()

    const slotFlags = () => {
        const ranked = ranking.rank(territories.flagNeeds())
        slots.show(ranked, (code) => {
            const region = regions.get(code)
            return region === undefined ? undefined : region.width / region.height
        })
        territories.markSlots(ranked)
    }

    const sphere = innerSphere()
    const picker = new GpuPicker(renderer, field.pickingPoints, sphere);
    const clock = new MapClock()
    const tiles = new MapTiles(field.size, TILES_PER_BATCH, clock)
    const ownership = tiles.ownership
    const shielded = tiles.shields
    const holdings = new Holdings()

    let country: Country = initialCountry;

    const bonusBox = createBonusBox()
    scene.add(bonusBox.object)

    const bonusPointer = createBonusPointer(eventTarget)

    const enclosures = createEnclosureEffects(geometryData.positions)
    scene.add(enclosures.object)

    const bonusClicks = createClickEffects(geometryData.positions)
    scene.add(bonusClicks.object)

    const plainClicks = createClickGlints(geometryData.positions)
    scene.add(plainClicks.object)

    const outline = createBorderLines(borderLines)
    scene.add(outline.object)

    let offered: BonusOffer | undefined

    const ownClicks = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownHits = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownPlacements = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownEnclosures = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)

    const driveBonusBox = (seconds: number) => {
        const enclosing = enclosures.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())
        const spreading = bonusClicks.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())
        const clicking = plainClicks.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())
        const boxed = bonusBox.update(seconds, camera)
        bonusPointer.update(bonusBox.flying ? bonusBox.object.position : undefined, camera)
        return enclosing || spreading || clicking || boxed
    }

    const targets = new FortifyTargets({
        sizeOf: (landmass) => territories.sizeOf(landmass),
        leaderOf: (landmass) => holdings.leaderOf(landmass),
        ownerOf: (tile) => ownership.ownerOf(tile),
        known: (tile) => tiles.known(tile),
        tilesOf: (landmass) => landmasses.tilesOf(landmass),
        landmassOf: (tile) => borders.assignment[tile - 1] ?? 0,
    }, (tiles, on) => {
        field.setPulses(tiles, on)
        invalidate()
    })
    // Off until the locks are read: without them a pulse would show for the flag that fortified last.
    let tracking = false

    const announceClose = (entered: Entered[]) => {
        for (const {landmass, target} of entered) {
            if (target.flag !== country.code) continue
            onCloseToFortify({landmass, name: landmassName(landmass, borders.codes[landmass]), missing: target.missing})
        }
    }

    const readFortresses = async () => {
        try {
            const locks = await ownershipsGetter.getFortresses(lifetime.signal)
            if (lifetime.signal.aborted) return
            targets.lockAll(locks, borders.codes.length)
            tracking = true
        } catch (e) {
            if (!lifetime.signal.aborted) console.error("could not read the fortresses", e)
        }
    }

    const pulsing = (camera: THREE.Camera) => {
        for (const landmass of targets.landmasses()) {
            const at = landmass * 5
            const along = camera.position.x * borders.frames[at] + camera.position.y * borders.frames[at + 1] + camera.position.z * borders.frames[at + 2]
            if (along > 0) return true
        }
        return false
    }

    const applyChanges = (changes: OwnerChange[], live = true) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        if (tracking) {
            const entered = targets.touch(changes.map(({tile}) => tile))
            if (live) announceClose(entered)
        }
        invalidate()
    }

    const showShields = (changes: ShieldChange[]) => {
        if (changes.length === 0) return
        field.setShields(changes)
        invalidate()
    }

    const noChanges = (): TileChanges => ({owners: [], shields: []})

    const recount = ({countries, landmasses}: Recount, live: boolean) => {
        if (landmasses.length > 0) {
            territories.setRows(landmasses, (landmass) => holdings.leaderOf(landmass))
            slotFlags()
            if (tracking) {
                const entered = targets.recount(landmasses)
                if (live) announceClose(entered)
            }
            invalidate()
        }
        if (countries) updateLeaderboard(rankCountries(holdings.counts()), live)
    }

    let holdingsAt: MapVersion | undefined

    const takeHoldings = (read: Counts) => {
        if (!clock.counts(read.version)) return
        if (!holdingsAt || holdingsAt.epoch !== read.version.epoch || holdingsAt.seq < read.version.seq) holdingsAt = read.version
        recount(holdings.read(read), false)
    }

    const holdingsCurrent = () => holdingsAt !== undefined && clock.current(holdingsAt)

    const playShields = (changes: ShieldChange[]) => {
        const seconds = performance.now() / 1000
        for (const {tile, shields, was} of changes) {
            if (shields < was && !ownHits.has(tile, country.code, seconds)) plainClicks.playHit(tile, camera)
            if (shields > was && !ownPlacements.has(tile, country.code, seconds)) plainClicks.playShielded(tile, camera)
        }
    }

    const blasts = createBlasts(uniforms, uniforms.pixelsPerRadian, uniforms.pixelRatio)
    scene.add(blasts.object)
    const blastPointer = createBonusPointer(eventTarget, "blast")

    let charges: Charges = NO_CHARGES

    let switches: Switches = ALL_OFF

    const switchTo = (next: Switches) => {
        if (next === switches) return
        switches = next
        onSwitchesChange(next)
    }

    let rules: BonusRules | undefined

    let armed: {radius: number} | undefined

    const hold = new HoldToDrop<THREE.Vector3>({holdSeconds: HOLD_TO_DROP_SECONDS, tolerancePx: HOLD_TOLERANCE_PX})

    let swallowClick = false

    const press = new ClickOrDrag(CLICK_TOLERANCE)

    const peek = new TilePeek(PEEK_RULES)
    const territoryPeek = new TilePeek(PEEK_RULES)
    let mouse = false
    let pointerAt: Point = {x: 0, y: 0}
    let peeking: number | undefined
    let peekingTerritory: number | undefined
    let hoveredLandmass: number | undefined
    let shownLandmass: number | undefined

    const landmassOf = (tile: number | undefined) => (tile === undefined ? 0 : borders.assignment[tile - 1]) || undefined

    const orbiting = () => inOrbit(camera.zoom, renderer.domElement.height / renderer.getPixelRatio())

    const peekAt = ({spot: tile, point, touch}: Peeked) => {
        const owner = ownership.ownerOf(tile)
        if (!owner) return
        peeking = tile
        onPeek({tile, countryCode: owner, point, touch})
    }

    const endPeek = () => {
        if (peeking === undefined) return
        peeking = undefined
        onPeekEnd()
    }

    const peekTerritoryAt = ({spot: landmass, point, touch}: Peeked) => {
        peekingTerritory = landmass
        onTerritoryPeek({landmass, point, touch})
    }

    const endTerritoryPeek = () => {
        if (peekingTerritory === undefined) return
        peekingTerritory = undefined
        onTerritoryPeekEnd()
    }

    let flight: Flight | undefined

    const flyTo = (landmass: number | undefined) => {
        if (landmass === undefined) return
        const {width, height} = renderer.domElement
        const {least, most} = flownZoomsOf(height / renderer.getPixelRatio())
        const points = Array.from(landmasses.tilesOf(landmass), (tile) => pointOf(geometryData.positions, tile))
        const to = territoryShotOf(points, width / height, least, Math.min(most, MAX_ZOOM))
        if (!to) return
        const {x, y, z} = camera.position.clone().normalize()
        flight = {
            from: {direction: {x, y, z}, zoom: camera.zoom},
            to,
            since: performance.now() / 1000,
            seconds: prefersReducedMotion() ? 0 : FLIGHT_SECONDS,
        }
    }

    const pilot: Pilot = {
        shotAt: (seconds) => {
            if (!flight) return undefined
            const share = flight.seconds > 0 ? (seconds - flight.since) / flight.seconds : 1
            const shot = flightAt(flight.from, flight.to, share)
            if (share >= 1) flight = undefined
            return shot
        },
        cancel: () => {
            flight = undefined
        },
    }

    const highlight = (orbit: boolean) => {
        const landmass = orbit ? hoveredLandmass ?? shownLandmass ?? 0 : 0
        if (uniforms.highlight.value === landmass) return false
        uniforms.highlight.value = landmass
        outline.setHighlight(landmass)
        return true
    }

    const cancelCharge = () => {
        hold.cancel()
        blasts.setCharge(0)
    }

    const disarm = () => {
        if (!armed) return
        armed = undefined
        cancelCharge()
        blasts.setAim(undefined, 0)
        eventTarget.classList.remove("viewer-canvas--armed")
        onArmedChange(false)
    }

    const arm = () => {
        if (armed || !charges.bomb || !bomber || !rules) return
        armed = {radius: rules.blastRadius}
        eventTarget.classList.add("viewer-canvas--armed")
        onArmedChange(true)
        switchTo(ALL_OFF)
    }

    const takeCharges = (held: Charges) => {
        charges = held
        if (!held.bomb) disarm()
        switchTo(switchesHeld(switches, held))
        onCharges(held)
    }

    const aimAt = (point: THREE.Vector3 | undefined) => {
        if (!armed) return
        blasts.setAim(point, armed.radius)
    }

    const takeReward = ({reward, charges: held}: ClaimedBonus) => {
        takeCharges(held)
        onBonusWon(reward)
    }

    // Subscribe only once everything it calls exists: a feed may answer at once.
    const stopBonuses = bonusListener?.listenForBonuses({
        onOffered: (offer) => {
            offered = offer
            bonusBox.spawn(offer.seed, (offer.expiresAt - CLAIM_MARGIN_MS) / 1000)
            playSound("bonusSpawn")
        },
        onTaken: (taken) => onBonusTaken(taken),
        onEnclosed: (enclosure) => {
            enclosures.play(enclosure)
            if (!enclosure.yours) return
            playSound("enclose")
            ownEnclosures.record(enclosure.closingTile, enclosure.countryId, performance.now() / 1000)
        },
        onSpread: (spread) => {
            bonusClicks.playSpread(spread)
            if (ownClicks.has(spread.tile, spread.countryId, performance.now() / 1000)) playSound("spread")
        },
        onCharges: (held) => takeCharges(held),
        onRules: (read) => {
            rules = read
            if (read.tileShields > 0 && read.tileShields !== uniforms.shieldMost.value) {
                uniforms.shieldMost.value = read.tileShields
                uniforms.shieldMarks.value.dispose()
                uniforms.shieldMarks.value = drawShieldMarks(read.tileShields, invalidate)
                uniforms.shieldCells.value = shieldCellsOf(read.tileShields)
            }
            invalidate()
            onRules(read)
        },
    })

    let ownDropAt: number | undefined

    const dropBomb = (point: THREE.Vector3) => {
        if (!bomber) return
        disarm()
        ownDropAt = performance.now() / 1000
        bomber.dropBomb({x: point.x, y: point.y, z: point.z}, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            ownDropAt = undefined
            reportClaimFailure(e, {onSessionUnavailable, onBanned})
        })
    }

    let pendingClears: {at: number, drop: BombDrop}[] = []

    // A later change of a tile wins over the clear, by its version, however long the clear waited.
    const land = ({cleared, struck, version}: BombDrop) => {
        const changes = noChanges()
        tiles.applyBlast(cleared, struck, version, changes)
        applyChanges(changes.owners)
        showShields(changes.shields)
        playShields(changes.shields)
    }

    const flushClears = (upTo: number) => {
        if (pendingClears.length === 0) return false
        const due = pendingClears.filter((clear) => clear.at <= upTo)
        if (due.length === 0) return false
        pendingClears = pendingClears.filter((clear) => clear.at > upTo)
        for (const {drop} of due) land(drop)
        return true
    }

    let shakeFrom: number | undefined
    const shake = new THREE.Vector3()

    const stopBombs = bomber?.listenForBombs((drop) => {
        tiles.heard(drop.version)
        const seconds = performance.now() / 1000
        const centre = new THREE.Vector3(drop.point.x, drop.point.y, drop.point.z)
        blasts.start(centre, drop.radius, seconds, drop.tile === undefined)

        if (document.hidden) {
            land(drop)
        } else if (drop.cleared.length > 0 || drop.struck.length > 0) {
            pendingClears.push({at: seconds + IMPACT_DELAY, drop})
        }

        const own = ownDropAt !== undefined && drop.countryId === country.code && seconds - ownDropAt < OWN_DROP_WINDOW_SECONDS
        if (own) {
            ownDropAt = undefined
            shakeFrom = seconds + IMPACT_DELAY
        }
        playSound("bomb", {volume: own ? 1 : DISTANT_BOMB_VOLUME, onWater: drop.tile === undefined})
        onBombDropped(drop, drop.tile === undefined ? undefined : countryOfTile(borders, drop.tile))
    })

    const placeShield = (tile: number) => {
        if (!shielder) return
        ownPlacements.record(tile, country.code, performance.now() / 1000)
        plainClicks.playShielded(tile, camera)
        playSound("click")

        shielder.placeShield(tile, country.code).catch((e) => {
            if (lifetime.signal.aborted || e instanceof ShieldRefusedError) return
            if (e instanceof AllegianceLockedError) onAllegianceLocked(e.allegiance)
            else reportClaimFailure(e, {onSessionUnavailable, onBanned})
        })
    }

    const driveBlasts = (seconds: number) => {
        let drawing = blasts.update(seconds, camera)
        blastPointer.update(blasts.newest(seconds), camera)
        if (flushClears(seconds)) drawing = true

        if (armed) {
            const {progress, drop} = hold.tick(seconds)
            if (drop) dropBomb(drop)
            else blasts.setCharge(progress)
        }

        if (shakeFrom !== undefined && uniforms.motion.value > 0) {
            const s = seconds - shakeFrom
            if (s > SHAKE_SECONDS) {
                shakeFrom = undefined
            } else if (s >= 0) {
                // Undone after the render, or OrbitControls reads it back as the player turning.
                const amplitude = 0.015 * (1 - s / SHAKE_SECONDS) ** 2 / camera.zoom
                shake.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).multiplyScalar(amplitude)
                camera.position.add(shake)
                drawing = true
            }
        }

        return drawing
    }

    const undoShake = () => {
        camera.position.sub(shake)
        shake.set(0, 0, 0)
    }

    const canvasPosition = (at: Pick<MouseEvent, "clientX" | "clientY">) => {
        const canvas = renderer.domElement
        const rect = canvas.getBoundingClientRect()
        return {
            x: (at.clientX - rect.left) * (canvas.width / rect.width),
            y: (at.clientY - rect.top) * (canvas.height / rect.height),
        }
    }

    const pointerNdc = new THREE.Vector2()
    const deviceCoordinates = (x: number, y: number) => {
        const canvas = renderer.domElement
        return pointerNdc.set((x / canvas.width) * 2 - 1, -(y / canvas.height) * 2 + 1)
    }

    const raycaster = new THREE.Raycaster()
    const globeSurface = new THREE.Sphere(new THREE.Vector3(), 1)

    const surfacePoint = (x: number, y: number) => {
        raycaster.setFromCamera(deviceCoordinates(x, y), camera)
        return raycaster.ray.intersectSphere(globeSurface, new THREE.Vector3()) ?? undefined
    }

    let pendingPointer: {x: number, y: number} | undefined
    let mouseOnCanvas: {x: number, y: number} | undefined
    let wasInOrbit = orbiting()

    eventTarget.addEventListener('mousemove', (event: MouseEvent) => {
        pendingPointer = mouseOnCanvas = canvasPosition(event)
        pointerAt = {x: event.clientX, y: event.clientY}
    }, listenerOptions);

    eventTarget.addEventListener('mouseleave', () => {
        pendingPointer = mouseOnCanvas = undefined
        field.setHover(undefined)
        peek.hover(undefined, 0, pointerAt)
        territoryPeek.hover(undefined, 0, pointerAt)
        hoveredLandmass = undefined
        endPeek()
        endTerritoryPeek()
        if (armed && !hold.holding) aimAt(undefined)
    }, listenerOptions);

    eventTarget.addEventListener('pointerdown', (event: PointerEvent) => {
        press.begin(event)
        endPeek()
        endTerritoryPeek()
        const pressed = !armed && event.pointerType !== "mouse" ? canvasPosition(event) : undefined
        const pressedTile = pressed && picker.pick(camera, pressed.x, pressed.y)
        const orbit = orbiting()
        peek.begin(event, orbit || !loaded ? undefined : pressedTile, performance.now() / 1000)
        territoryPeek.begin(event, orbit ? landmassOf(pressedTile) : undefined, performance.now() / 1000)
        if (!loaded || !armed || !event.isTrusted || !event.isPrimary || event.button !== 0) return

        const {x, y} = canvasPosition(event)
        const point = surfacePoint(x, y)
        if (!point) return

        hold.begin(event.pointerId, event.clientX, event.clientY, performance.now() / 1000, point)
        swallowClick = true
        aimAt(point)
    }, listenerOptions);

    window.addEventListener('keydown', (event: KeyboardEvent) => {
        if (event.key === "Escape") disarm()
    }, listenerOptions);

    eventTarget.addEventListener('pointermove', (event: PointerEvent) => {
        press.move(event)
        hold.move(event.pointerId, event.clientX, event.clientY)
        peek.move(event)
        territoryPeek.move(event)
        mouse = event.pointerType === "mouse"
    }, listenerOptions);

    for (const ending of ['pointerup', 'pointercancel'] as const) {
        eventTarget.addEventListener(ending, (event: PointerEvent) => {
            hold.end(event.pointerId)
            peek.end(event.pointerId)
            territoryPeek.end(event.pointerId)
        }, listenerOptions);
    }

    eventTarget.addEventListener('pointerup', (event: PointerEvent) => press.end(event), listenerOptions);

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        if (!event.isTrusted) return;

        if (swallowClick) {
            swallowClick = false
            return
        }

        if (peek.peeked || territoryPeek.peeked) return

        const spot = press.spotOf(event)
        if (!spot) return

        const {x, y} = canvasPosition(spot)

        if (offered && monotonicNow() >= offered.expiresAt - CLAIM_MARGIN_MS) {
            offered = undefined
            bonusBox.hide()
            invalidate()
        }

        if (bonusBox.hitTest(camera, deviceCoordinates(x, y)) && bonusBox.take()) {
            const claimed = offered
            offered = undefined
            if (!claimed) return
            playSound("bonusCaught")

            bonusListener?.claimBonus(claimed.token, country.code)
                .then(takeReward)
                .catch((e) => {
                    if (lifetime.signal.aborted) return
                    reportClaimFailure(e, {onSessionUnavailable, onBanned})
                })
            return
        }

        if (armed) return

        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return
        peek.clicked()
        territoryPeek.clicked()

        if (orbiting()) {
            flyTo(landmassOf(tile))
            return
        }

        if (!loaded) return

        if (!regions.get(country.code)) {
            warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
            return
        }

        const owner = ownership.ownerOf(tile)
        const shields = shielded.shieldsOf(tile)

        if (switches.shield && shielder) {
            const placement = placementOf(owner, country.code, shields, rules?.tileShields)
            if (placement === "full") {
                onNotice("shieldFull")
                return
            }
            if (placement === "place") {
                placeShield(tile)
                return
            }
        }

        if (!canClick(budget, monotonicNow())) {
            onRateLimited()
            playSound("refused")
            return
        }

        const outcome = outcomeOf(owner, country.code, shields)
        const {changes, claim} = outcome === "shielded"
            ? {changes: [], claim: undefined}
            : ownership.applyOptimistic(tile, country.code)
        applyChanges(changes)
        playSound("click")

        const seconds = performance.now() / 1000
        ownClicks.record(tile, country.code, seconds)
        if (outcome === "shielded") {
            ownHits.record(tile, country.code, seconds)
            plainClicks.playHit(tile, camera)
        } else {
            plainClicks.playOwnClick(tile, camera)
        }

        const clicked = country.code
        const shielding = switches.shield && shielder !== undefined
        const enclosing = switches.enclose && outcome === "taken"
        tileClicker.clickTile(tile, clicked, switches).then(() => {
            if (lifetime.signal.aborted) return
            onClickAccepted({country: clicked, took: outcome === "taken"})
            if (shielding) onNotice(outcome === "taken" ? "shieldTaken" : "shieldNotYours")
            if (enclosing) setTimeout(() => {
                if (lifetime.signal.aborted || ownEnclosures.has(tile, clicked, performance.now() / 1000)) return
                onNotice("nothingEnclosed")
            }, ENCLOSURE_WAIT_MS)
        }, (e) => {
            if (lifetime.signal.aborted) return
            applyChanges(ownership.rollback(claim))
            if (reportClickFailure(e, {onRateLimited, onVPNBlocked, onBanned, onSessionUnavailable, onAllegianceLocked})) playSound("refused")
        })
    }, listenerOptions);

    const resizeListener = () => {
        const {width, height} = layoutViewport();

        camera.left = -cameraSize * (width / height);
        camera.right = cameraSize * (width / height);
        camera.updateProjectionMatrix();

        renderer.setPixelRatio(pixelRatio(graphics));
        renderer.setSize(width, height);
        invalidate();
    };
    window.addEventListener('resize', resizeListener, listenerOptions);

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch((updates: Update[]) => {
        const changes = noChanges()
        tiles.applyUpdates(updates, changes)
        applyChanges(changes.owners)
        showShields(changes.shields)
        playShields(changes.shields)

        const seconds = performance.now() / 1000
        for (const {tile, clicked, previousCountry, newCountry} of updates) {
            if (!clicked || previousCountry === newCountry || ownClicks.has(tile, country.code, seconds)) continue
            plainClicks.playClick(tile, camera)
        }
    })

    // The counts go out once a second: they are current only up to the last change the stream told of them.
    let holdingsTold: MapVersion | undefined

    const stopHoldings = updatesListener.listenForHoldings((change) => {
        tiles.heard(change.version)
        if (!clock.counts(change.version)) return
        holdingsTold = change.version
        recount(holdings.change(change), true)
    })

    let tilesWereRead: () => void = () => {}
    const tilesRead = new Promise<void>((resolve) => {
        tilesWereRead = resolve
    })

    const takeBatch = (batch: Ownerships) => {
        const changes = noChanges()
        tiles.read(batch, changes)
        applyChanges(changes.owners, false)
        showShields(changes.shields)
        if (tracking) targets.touch(rangeTiles(batch.range))
        if (!loaded) {
            const share = tiles.readShare()
            onLoadProgress(share)
            if (share === 1) {
                loaded = true
                tilesWereRead()
            }
        }
    }

    let reading: AbortController | undefined
    let firstOpened: (() => void) | undefined
    const opened = new Promise<void>((resolve) => {
        firstOpened = resolve
    })

    // Reads again, every READ_AGAIN_MS and a few times at most, what comes back older than the stream's opening.
    const catchUp = async (ranges: readonly TileRange[], holdingsToo: boolean, locksToo: boolean) => {
        reading?.abort()
        const round = new AbortController()
        reading = round

        let stale = ranges
        let staleHoldings = holdingsToo
        try {
            for (let attempt = 0; attempt < READS_AT_MOST && (stale.length > 0 || staleHoldings); attempt++) {
                if (attempt > 0) {
                    await pause(READ_AGAIN_MS, round.signal)
                    stale = tiles.stale(stale)
                    staleHoldings = !holdingsCurrent()
                }
                await Promise.all([
                    staleHoldings ? ownershipsGetter.getHoldings(round.signal).then((read) => {
                        if (!round.signal.aborted) takeHoldings(read)
                    }) : undefined,
                    stale.length > 0 ? ownershipsGetter.getOwnerships(stale, takeBatch, round.signal) : undefined,
                    locksToo && attempt === 0 ? readFortresses() : undefined,
                ])
                await Promise.race([opened, aborted(round.signal)])
                stale = tiles.stale(stale)
                staleHoldings = !holdingsCurrent()
            }
        } catch (e) {
            if (round.signal.aborted) return
            console.error("could not read the map", e)
            if (!loaded) void pause(READ_AGAIN_MS, lifetime.signal).then(() => catchUp(tiles.stale(tiles.ranges), false, false), () => {})
        }
    }

    // A stream that opens again had a gap, and nothing replays it: what it may have missed is read again at once.
    const stopOpenings = updatesListener.listenForOpenings((version) => {
        if (holdingsCurrent() && holdingsTold && holdingsAt && holdingsTold.epoch === holdingsAt.epoch && holdingsTold.seq > holdingsAt.seq) holdingsAt = holdingsTold
        const {restarted} = tiles.opened(version)
        if (restarted) holdings.forget()
        if (firstOpened) {
            firstOpened()
            firstOpened = undefined
            return
        }
        void catchUp(tiles.stale(tiles.ranges), !holdingsCurrent(), true)
    })

    const waves = createFortifyWaves(uniforms, geometryData.positions, (tiles) => {
        field.setShields(Array.from(tiles, (tile) => ({tile, shields: shielded.shieldsOf(tile), was: 0})))
        invalidate()
    })

    const stopFortifications = updatesListener.listenForFortifications((fortification) => {
        const members = landmasses.tilesOf(fortification.landmass)
        const raised = tiles.applyFortification(members, fortification.countryId, uniforms.shieldMost.value, fortification.version)
        if (document.hidden || uniforms.motion.value === 0) showShields(raised)
        else waves.start(fortification.landmass, members, fortification.tile, performance.now() / 1000)

        const own = fortification.countryId === country.code
        const news = isNews(fortification, members.length, country.code)
        if (news) playSound("fortify", {volume: own ? 1 : DISTANT_BOMB_VOLUME})
        onFortified({fortification, name: landmassName(fortification.landmass, borders.codes[fortification.landmass]), tiles: members.length, news})
        if (tracking) targets.lock(fortification.landmass, fortification.countryId)
    })

    addDisplayObjects(scene, field.displayPoints, graphics, sphere)

    let captureRequests: CaptureRequest[] = []

    const takeCaptureRequests = () => {
        const waiting = captureRequests
        captureRequests = []
        return waiting
    }

    const {stop: stopAnimation} = startAnimation(renderer, scene, camera, uniforms, pickingUniforms, director, pilot, () => mapView, () => {
        const was = dirty
        dirty = false
        return was
    }, (seconds) => {
        const boxed = driveBonusBox(seconds)
        const fortifying = waves.update(seconds)
        const blasting = driveBlasts(seconds) || fortifying
        outline.update(camera.zoom, renderer.domElement.width, renderer.domElement.height, renderer.getPixelRatio(), mapView)

        const peeked = armed ? undefined : peek.tick(seconds)
        if (peeked !== undefined) peekAt(peeked)
        const peekedTerritory = armed ? undefined : territoryPeek.tick(seconds)
        if (peekedTerritory !== undefined) peekTerritoryAt(peekedTerritory)

        const orbit = orbiting()
        if (orbit !== wasInOrbit) {
            wasInOrbit = orbit
            pendingPointer ??= mouseOnCanvas
            onOrbitChange(orbit)
        }
        const lit = highlight(orbit)

        if (pendingPointer === undefined) return boxed || blasting || lit
        const {x, y} = pendingPointer
        pendingPointer = undefined
        if (armed) {
            field.setHover(undefined)
            hoveredLandmass = undefined
            highlight(orbit)
            if (!hold.holding) aimAt(surfacePoint(x, y))
            return true
        }
        const hovered = picker.pick(camera, x, y)
        if (mouse) {
            const tile = orbit ? undefined : hovered
            const landmass = orbit ? landmassOf(hovered) : undefined
            peek.hover(tile, seconds, pointerAt)
            territoryPeek.hover(landmass, seconds, pointerAt)
            if (tile !== peeking) endPeek()
            if (landmass !== peekingTerritory) endTerritoryPeek()
            hoveredLandmass = landmass
        }
        const relit = highlight(orbit)
        return field.setHover(hovered) || relit || lit || boxed || blasting
    }, () => {
        undoShake()
        if (captureRequests.length === 0) return

        const frame = readDrawingBuffer(renderer)
        for (const request of takeCaptureRequests()) request.resolve(frame)
    }, pulsing);

    const globe: Globe = {
        tilesCount: field.size,
        tilesRead,
        setCountry: (newCountry: Country) => {
            country = newCountry
        },
        setMapView: (view: MapView) => {
            if (view === mapView) return
            mapView = view
            invalidate()
        },
        takeReward,
        setArmed: (on: boolean) => on ? arm() : disarm(),
        setSwitch: (name: keyof Switches, on: boolean) => {
            const next = switchesHeld(switched(switches, name, on), charges)
            if (next[name]) disarm()
            switchTo(next)
        },
        setClickHue: (hue: number | undefined) => plainClicks.setOwnHue(hue),
        territoryOf: (landmass: number) => ({
            landmass,
            name: landmassName(landmass, borders.codes[landmass]),
            ground: borders.codes[landmass] || undefined,
            size: territories.sizeOf(landmass),
            holdings: rankHoldings(holdings.holdingsOf(landmass)),
            fortifiedBy: targets.lockOf(landmass),
            target: targets.targetOf(landmass),
        }),
        showTerritory: (landmass: number | undefined) => {
            shownLandmass = landmass
        },
        capture: () => new Promise<CapturedFrame>((resolve, reject) => {
            if (lifetime.signal.aborted) {
                reject(new Error("the globe is no longer running"))
                return
            }
            captureRequests.push({resolve, reject})
            invalidate()
        }),
        graphics: () => graphicsReportOf(renderer.getContext()),
        dispose: () => {
            lifetime.abort()

            for (const request of takeCaptureRequests()) {
                request.reject(new Error("the globe was disposed before the frame was read"))
            }

            stopAnimation()
            cleanUpdatesListener()
            stopOpenings()
            stopHoldings()
            stopFortifications()
            reading?.abort()
            stopBonuses?.()
            stopBombs?.()
            stopWatchingBudget?.()

            picker.dispose()
            bonusPointer.dispose()
            blastPointer.dispose()
            blasts.dispose()
            field.dispose()
            territories.dispose()
            slots.dispose()
            outline.dispose()
            bonusBox.dispose()
            enclosures.dispose()
            bonusClicks.dispose()
            plainClicks.dispose()

            cleanup()
        }
    }

    const stopFollowingLoad = new AbortController()
    signal.addEventListener("abort", () => lifetime.abort(), {signal: stopFollowingLoad.signal})
    onLoadProgress(0)
    void catchUp(tiles.ranges, false, false)
    try {
        const [read] = await Promise.all([ownershipsGetter.getHoldings(lifetime.signal), readFortresses()])
        lifetime.signal.throwIfAborted()
        takeHoldings(read)
    } catch (e) {
        globe.dispose()
        throw e
    } finally {
        stopFollowingLoad.abort()
    }

    onOrbitChange(wasInOrbit)
    return globe
}

function rangeTiles({start, end}: TileRange): number[] {
    return Array.from({length: end - start + 1}, (_, i) => start + i)
}

function startAnimation(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.OrthographicCamera,
    uniforms: Uniforms,
    pickingUniforms: {pointSize: THREE.IUniform},
    director: Director | undefined,
    pilot: Pilot,
    mapView: () => MapView,
    takeChange: () => boolean,
    beforeRender: (seconds: number) => boolean,
    afterRender: () => void,
    pulsing: (camera: THREE.Camera) => boolean,
): {stop: () => void} {
    const starfield = createStarfield();

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.minZoom = MIN_ZOOM;
    controls.maxZoom = MAX_ZOOM;
    controls.panSpeed = 0.1;
    controls.enableDamping = true;
    controls.autoRotateSpeed = SPIN_TURNS_PER_MINUTE;
    controls.enabled = director === undefined;

    // A wheel zoom moves the camera inside OrbitControls' own handler, so update() misses it.
    let moved = false;

    controls.addEventListener('change', () => {
        moved = true;

        controls.autoRotate = camera.zoom <= RESTING_ZOOM;

        controls.rotateSpeed = (1 / camera.zoom) / 1.5;
    });

    let interactingUntil = 0;
    controls.addEventListener('start', () => {
        interactingUntil = Infinity;
        pilot.cancel();
    });
    controls.addEventListener('end', () => {
        interactingUntil = performance.now() + INTERACTION_GRACE_MS;
    });

    let drawnAt = -Infinity;
    let tickedAt: number | undefined;
    const viewport = new THREE.Vector2();

    renderer.setAnimationLoop((time: number) => {
        const sinceLastTick = tickedAt === undefined ? 0 : time - tickedAt;
        tickedAt = time;

        let turned = director
            ? aim(camera, director(time / 1000))
            : controls.update(spinStep(sinceLastTick)) || moved;

        const flown = director ? undefined : pilot.shotAt(time / 1000);
        if (flown) {
            // Over whatever the controls did this tick: their spin and their glide die out under the flight.
            controls.target.set(0, 0, 0);
            turned = aim(camera, flown);
            interactingUntil = Math.max(interactingUntil, performance.now() + INTERACTION_GRACE_MS);
        }

        const {y: height} = renderer.getSize(viewport);
        const ratio = renderer.getPixelRatio();
        uniforms.pointSize.value = displayPointSize(camera.zoom, height, mapView()) * ratio;
        pickingUniforms.pointSize.value = pickPointSize(camera.zoom, height) * ratio;
        uniforms.pixelRatio.value = ratio;

        uniforms.flagPaint.value = flagPaint(camera.zoom, height, mapView());
        uniforms.pixelsPerRadian.value = (renderer.domElement.height / 2) * camera.zoom;

        const moving = beforeRender(time / 1000);
        const changed = takeChange() || moving;

        const ambient = pulsing(camera);
        if (!drawsFrame({turned, changed, ambient, at: time, drawnAt, interactingUntil, sinceLastTick})) return;

        moved = false;
        drawnAt = time;

        starfield.render(renderer, camera, () => renderer.render(scene, camera));
        afterRender();
    });

    return {
        stop: () => {
            renderer.setAnimationLoop(null);
            controls.dispose();
            starfield.dispose();
        },
    };
}

function aim(camera: THREE.OrthographicCamera, {direction, zoom}: Shot): boolean {
    const distance = camera.position.length()
    camera.position.set(direction.x, direction.y, direction.z).setLength(distance)
    camera.lookAt(0, 0, 0)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    return true
}

function prefersReducedMotion(): boolean {
    return typeof window.matchMedia === "function"
        && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

export function reportClaimFailure(
    error: unknown,
    handlers: {onSessionUnavailable: () => void, onBanned: (until: number) => void},
) {
    if (error instanceof BonusLostError) return
    if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else if (error instanceof BannedError) handlers.onBanned(error.until)
    else console.error(error)
}

export function reportClickFailure(
    error: unknown,
    handlers: {
        onRateLimited: () => void,
        onVPNBlocked: () => void,
        onBanned: (until: number) => void,
        onSessionUnavailable: () => void,
        onAllegianceLocked: (allegiance: Allegiance) => void,
    },
): boolean {
    if (error instanceof RateLimitedError) handlers.onRateLimited()
    else if (error instanceof VPNBlockedError) handlers.onVPNBlocked()
    else if (error instanceof BannedError) handlers.onBanned(error.until)
    else if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else if (error instanceof AllegianceLockedError) handlers.onAllegianceLocked(error.allegiance)
    else {
        console.error(error)
        return false
    }
    return true
}

function aborted(signal: AbortSignal): Promise<never> {
    return new Promise((_, reject) => signal.addEventListener("abort", () => reject(signal.reason), {once: true}))
}

function pause(ms: number, signal: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
        const timer = setTimeout(resolve, ms)
        signal.addEventListener("abort", () => {
            clearTimeout(timer)
            reject(signal.reason)
        }, {once: true})
    })
}
