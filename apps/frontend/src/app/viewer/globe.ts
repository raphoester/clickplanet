import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, setupScene} from "./scene.ts";
import {Sharpness} from "./sharpness.ts";
import {endStrayFingers} from "./strayFingers.ts";
import {Halo, paintHalo} from "./atmosphere.ts";
import {graphicsOf} from "./graphics.ts";
import {loadPointGeometryData} from "./points.ts";
import {TilePicker} from "./tilePicker.ts";
import {innerSphere} from "./sphere.ts";
import {CapturedFrame, readDrawingBuffer} from "./capture.ts";
import {GraphicsReport, graphicsReportOf} from "./graphicsReport.ts";
import {TileField} from "./tileField.ts";
import {BorderField, leastPaintedTiles, loadBorders, loadProvinces} from "./borderField.ts";
import {coarseTolerancesOf, createBorderLines, loadBorderLines, loadProvinceLines} from "./borderLines.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import type {BoardAsset} from "../../domain/board.ts";
import {BoardScale, scaleOf} from "../../domain/boardScale.ts";
import {Capitals, loadCapitals} from "./capitals.ts";
import {loadBridges} from "./bridges.ts";
import {type BridgeArcs, createBridgeArcs} from "./bridgeArcs.ts";
import {
    Divisions,
    divisionsAt,
    flagPaint,
    flownZoomsOf,
    inOrbit,
    leastFlagRadiusOf,
    provinceShareAfter,
    tilePointSize,
    virusLiftOf,
} from "./pointSize.ts";
import {createTerritoryLayer} from "./territoryLayer.ts";
import {loadTerrain} from "./terrain.ts";
import {regions} from "./atlas.ts";
import {FlagSlots} from "./flagSlots.ts";
import {slotBudgetOf} from "./slotCells.ts";
import {LandmassSight, type ScreenView, tileAreaOf} from "./slotNeed.ts";
import {Settling, type Viewpoint} from "./settling.ts";
import {Country} from "../../domain/countries.ts";
import {Allegiance} from "../../domain/allegiance.ts";
import {MapView, Rendering} from "../../domain/displaySettings.ts";
import {
    AllegianceLockedError,
    BannedError,
    BombDrop,
    Bomber,
    GlobePoint,
    Counts,
    MapVersion,
    BonusListener,
    BombRefusedError,
    BonusLostError,
    BoxCatcher,
    BoxOffer,
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
    FrozenError,
    OutOfReachError,
    Infector,
    VirusRefusedError,
    Miner,
    MineRefusedError,
    Rallier,
    RallyRefusedError,
} from "../../backends/backend.ts";
import {clickCostOf, Rallies, WHOLE_CLICK} from "../../domain/rallies.ts";
import {createRallyMarks, RallyCallers} from "./rallyMarks.ts";
import {createRallyPops} from "./rallyPops.ts";
import {InfectionChange, infectionMarksOf} from "../../domain/infections.ts";
import {layingOf, MineChange, mineMarksOf} from "../../domain/mines.ts";
import {SessionUnavailableError} from "../../backends/session.ts";
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts";
import {OwnerChange} from "../../domain/tileOwnership.ts";
import {MapTiles, noTileChanges} from "../../domain/mapTiles.ts";
import {MapClock} from "../../domain/mapClock.ts";
import {Holdings, Recount} from "../../domain/holdings.ts";
import {warnOnce} from "../../domain/warnOnce.ts";
import {layoutViewport} from "./viewport.ts";
import {createStarfield} from "./stars.ts";
import {maxZoomOf, MIN_ZOOM, RESTING_ZOOM} from "./zoom.ts";
import {createBonusBox} from "./bonusBox.ts";
import {createBonusPointer} from "./bonusPointer.ts";
import {createEnclosureEffects} from "./enclosureEffect.ts";
import {createClickEffects} from "./clickEffects.ts";
import {createClickGlints, type Glints} from "./clickGlints.ts";
import {ALL_OFF, atRest, BonusNotice, BonusReward, BonusRules, Charges, NO_CHARGES, restingSwitch, sameSwitches, Switches, switched, switchesHeld} from "../../domain/bonus.ts";
import {canClick, ClickBudget, ClickBudgetSource, now as monotonicNow} from "../../backends/clickBudget.ts";
import {BlastUniforms, blastUniforms, createBlasts} from "./blasts.ts";
import {BLAST_TIMELINE, IMPACT_DELAY, mineBlastRadius} from "../../domain/blast.ts";
import {HoldToDrop} from "../../domain/holdToDrop.ts";
import {ClickOrDrag} from "../../domain/clickOrDrag.ts";
import {Peek, Peeked, Point, TilePeek} from "../../domain/tilePeek.ts";
import {type Place, placeKey, rankHoldings, Territory, TerritoryPeek} from "../../domain/territory.ts";
import {FLIGHT_SECONDS, flightAt, territoryShotOf} from "../../domain/flight.ts";
import {pointOf} from "../../domain/clip/geometry.ts";
import {Front, FrontChanges} from "../../domain/front.ts";
import {OpenGround, openTileNear, opensOn} from "../../domain/openTiles.ts";
import {createTapPointer} from "./tapPointer.ts";
import {CapitalMarks} from "../../domain/capitalMarks.ts";
import {Standing, standingOf} from "../../domain/standing.ts";
import {acrossBridges, type Bridges} from "../../domain/bridges.ts";
import {MissionGround, MissionMarks, MissionTarget, groundsOf, nearestPlayable} from "../../domain/missionTargets.ts";
import {Neighbours} from "../../domain/neighbours.ts";
import {GhostTake, Ghosts} from "../../domain/ghosts.ts";
import {GhostArmy, Ground, Sight} from "../../domain/ghostArmy.ts";
import {saveGhostTakes, savedGhostTakes} from "./ghostStore.ts";
import {OwnClicks} from "../../domain/ownClicks.ts";
import {ShieldChange, outcomeOf, placementOf} from "../../domain/shields.ts";
import {Landmasses} from "../../domain/landmasses.ts";
import {translator} from "../../i18n/page.ts";
import {Fortified, isNews} from "../../domain/fortify.ts";
import {FortifyTargets} from "../../domain/fortifyTargets.ts";
import {createFortifyWaves, FortifyUniforms, fortifyUniforms} from "./fortifyWaves.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import {drawShieldMarks, shieldCellsOf} from "./shieldMarks.ts";
import {AcceptedClick} from "../clicks/acceptedClicks.ts";
import {fileHashOf} from "../../domain/provinceLinesFile.ts";
import {NO_PROVINCES, ProvinceSpans} from "../../domain/provinces.ts";

type Uniforms = BlastUniforms & FortifyUniforms & {
    pointSize: THREE.IUniform
    // How far an infected tile is drawn nearer the camera, and the least half-height of a painted flag: in the board's
    // own spacings.
    virusLift: THREE.IUniform<number>
    leastFlagRadius: THREE.IUniform<number>
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
    // The rows of the grounds the flags are painted over, the landmasses' and the provinces', and how far the provinces
    // divide the map: 0 or 1, and in between while the one fades into the other.
    landmassData: THREE.IUniform<THREE.DataTexture>
    landmassCount: THREE.IUniform<number>
    provinceData: THREE.IUniform<THREE.DataTexture>
    provinceCount: THREE.IUniform<number>
    provinceShare: THREE.IUniform<number>
    pixelsPerRadian: THREE.IUniform<number>
    pixelRatio: THREE.IUniform<number>
    flagPaint: THREE.IUniform
    splitFlags: THREE.IUniform<number>
    shieldMost: THREE.IUniform<number>
    shieldMarks: THREE.IUniform<THREE.Texture>
    shieldCells: THREE.IUniform<THREE.Vector2>
    // The provinces lit, first and last.
    highlight: THREE.IUniform<THREE.Vector2>
    slotTexture: THREE.IUniform<THREE.Texture | null>
    slotFrames: THREE.IUniform<THREE.DataTexture>
    pulseTime: THREE.IUniform<number>
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

// A pulse that only says "look here", and a glint on a page nobody touches, are drawn at 20 frames a second.
const AMBIENT_FRAME_MS = 50

const SPIN_TURNS_PER_MINUTE = 2

const MAX_SPIN_STEP_MS = 100

export function spinStep(sinceLastTick: number): number {
    return Math.min(Math.max(sinceLastTick, 0), MAX_SPIN_STEP_MS) / 1000
}

const WATCHED_FOR_MS = 60_000

const INPUTS = ['pointerdown', 'pointermove', 'wheel', 'keydown', 'touchstart'] as const

export type Attention = {
    lastInputAt: number
    now: number
}

export function watched({lastInputAt, now}: Attention): boolean {
    return now - lastInputAt < WATCHED_FOR_MS
}

export type Spin = Attention & {
    zoom: number
}

export function spins({zoom, ...attention}: Spin): boolean {
    return zoom <= RESTING_ZOOM && watched(attention)
}

// The shaders' beat (display/vertex.glsl's PULSE_RATE): faintest at three quarters of each period.
const BEAT_SECONDS = 1.1

// A clock held on a trough is one only to rounding: without the slack it would creep on to the next.
const TROUGH_SLACK = 1e-9

export type Beat = {
    clock: number
    step: number
    attended: boolean
}

export function beatClock({clock, step, attended}: Beat): number {
    if (attended) return clock + step
    return Math.min(clock + step, nextTrough(clock))
}

function nextTrough(clock: number): number {
    return (Math.ceil(clock / BEAT_SECONDS - 0.75 - TROUGH_SLACK) + 0.75) * BEAT_SECONDS
}

const INTERACTION_GRACE_MS = 1_000

export type Tick = {
    turned: boolean
    changed: boolean
    glints: Glints
    ambient: boolean
    attended: boolean
    at: number
    drawnAt: number
    interactingUntil: number
    sinceLastTick: number
}

export function drawsFrame(
    {turned, changed, glints, ambient, attended, at, drawnAt, interactingUntil, sinceLastTick}: Tick,
): boolean {
    const atOnce = changed || glints === "over" || (glints === "playing" && attended)
    const paced = ambient || glints === "playing"
    if (!turned && !atOnce && !paced) return false
    if (atOnce || at <= interactingUntil) return true
    return at + sinceLastTick / 2 >= drawnAt + (turned ? IDLE_FRAME_MS : AMBIENT_FRAME_MS)
}

const CLAIM_MARGIN_MS = 2_000

// The ghosts raid only while fewer play than this.
const GHOSTS_BELOW = 7

const GHOSTS_EVERY_SECONDS = 0.2

// The share of the screen's shorter half, around its middle, that the ghosts raid: clear of the zones on its edges.
const GHOSTS_SIGHT = 0.7

const TAP_AIM_EVERY_SECONDS = 1

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
    // The board the globe draws: every tile id the backends hand it is a tile of it. The game page's is the one GetMap
    // names, read as the page loads.
    board: BoardAsset | Promise<BoardAsset>
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    clickBudget?: ClickBudgetSource
    container: HTMLElement
    linkedShot?: Shot
    // A first visit: the camera starts on the flag's own country, flies to its front once the tiles are read, and the front is lit.
    showFront?: boolean
    // A first visit before its first take: a hand taps a tile one click takes, until a take is accepted.
    showTap?: boolean
    country: Country
    mapView: MapView
    // The provinces divide the map once zoomed in close enough (divisionsAt in pointSize.ts), on a board that has them:
    // the play page. The video, the archive and the clips keep the countries at every zoom.
    provinces: boolean
    rendering: Rendering
    onLeaderboardChange: (entries: LeaderboardEntry[], live: boolean) => void
    // The share of the tiles read, and 1 once the frame that draws them all is drawn: a tile is clicked only then.
    onLoadProgress: (share: number) => void
    onRateLimited: () => void
    // A click on a frozen map, refused here or by the server.
    onFrozen: () => void
    onVPNBlocked: () => void
    onBanned: (until: number) => void
    onSessionUnavailable: () => void
    onAllegianceLocked: (allegiance: Allegiance) => void
    // A click on a tile the flag cannot reach while the reach plays, refused here or by the server.
    onOutOfReach: () => void
    // The flags while the knock-out plays, each time the rules, a map read or a knock-out may change them; undefined
    // while it does not.
    onStanding?: (standing: Standing | undefined) => void
    onBonusWon: (reward: BonusReward) => void
    onCharges: (charges: Charges) => void
    onRules: (rules: BonusRules) => void
    onSwitchesChange: (switches: Switches) => void
    bonusListener?: BonusListener
    boxCatcher?: BoxCatcher
    bomber?: Bomber
    onBombDropped: (drop: BombDrop, land: string | undefined) => void
    onArmedChange: (armed: boolean) => void
    infector?: Infector
    onVirusAimChange?: (aimed: boolean) => void
    miner?: Miner
    onMineAimChange?: (aimed: boolean) => void
    rallier?: Rallier
    onRallyAimChange?: (aimed: boolean) => void
    // Who called each rally: the name its flag on the globe carries.
    rallyCallers?: RallyCallers
    shielder?: Shielder
    onNotice?: (notice: BonusNotice) => void
    onClickAccepted?: (click: AcceptedClick) => void
    onFortified?: (fortified: Fortified) => void
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
    setCrowd(count: number | undefined): void
    // A battle runs: no ghost raid, and the ghost tiles go back.
    setBattle(on: boolean): void
    setMapView(view: MapView): void
    setHalo(halo: Halo): void
    takeReward(claimed: ClaimedBonus): void
    setArmed(armed: boolean): void
    setVirusAimed(aimed: boolean): void
    setMineAimed(aimed: boolean): void
    setRallyAimed(aimed: boolean): void
    setSwitch(name: keyof Switches, on: boolean): void
    setClickHue(hue: number | undefined): void
    // A territory seen from orbit, and the place it is given in it.
    territoryOf(place: Place): Territory
    showTerritory(place: Place | undefined): void
    flyToLastLoss(): void
    setMissions(targets: readonly MissionTarget[]): void
    flyToMission(target: MissionTarget): void
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
        linkedShot,
        showFront = false,
        showTap = false,
        country: initialCountry,
        mapView: initialMapView,
        provinces: byZoom,
        rendering,
        onLeaderboardChange: updateLeaderboard,
        onLoadProgress,
        onRateLimited,
        onFrozen,
        onVPNBlocked,
        onBanned,
        onSessionUnavailable,
        onAllegianceLocked,
        onOutOfReach,
        onStanding = () => {},
        onBonusWon,
        onCharges,
        onRules,
        onSwitchesChange,
        bonusListener,
        boxCatcher,
        bomber,
        onBombDropped,
        onArmedChange,
        infector,
        onVirusAimChange = () => {},
        miner,
        onMineAimChange = () => {},
        rallier,
        onRallyAimChange = () => {},
        rallyCallers,
        shielder,
        onNotice = () => {},
        onClickAccepted = () => {},
        onFortified = () => {},
        onPeek = () => {},
        onPeekEnd = () => {},
        onTerritoryPeek = () => {},
        onTerritoryPeekEnd = () => {},
        playSound = () => {},
        director,
        signal,
    } = options

    const graphics = graphicsOf(window.location.search, rendering);
    const sharpness = new Sharpness(graphics, director === undefined, signal);
    const board = await options.board;
    if (signal.aborted) throw new DOMException("globe load aborted", "AbortError");
    const bordersRead = loadBorders(board.urls.borders, signal)
    const [geometryData, borders, provinces, borderLines, terrain] = await Promise.all([
        loadPointGeometryData(board, signal),
        bordersRead,
        loadProvinces(board, bordersRead, signal),
        loadBorderLines(board.urls.borderLines, coarseTolerancesOf(scaleOf(board)), signal),
        loadTerrain(board.urls.terrain, board.lattice.cols, signal),
    ]);
    if (signal.aborted) throw new DOMException("globe load aborted", "AbortError");

    const lifetime = new AbortController();
    const listenerOptions = {signal: lifetime.signal};
    const scale = scaleOf(board);
    const maxZoom = maxZoomOf(scale);

    let budget: ClickBudget | undefined
    const stopWatchingBudget = clickBudget?.watchClickBudget((reading) => {
        budget = reading
    })

    let loaded = false
    let drawingWhole = false

    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget, graphics, sharpness.ratio());
    endStrayFingers(renderer.domElement, lifetime.signal)
    if (linkedShot) aim(camera, {...linkedShot, zoom: Math.min(maxZoom, Math.max(MIN_ZOOM, linkedShot.zoom))})
    let mapView: MapView = initialMapView
    const divides = byZoom && board.files.provinces !== undefined
    const divisionsNow = (shown: Divisions): Divisions => divides
        ? divisionsAt(camera.zoom, renderer.domElement.height / renderer.getPixelRatio(), shown)
        : "countries"
    let divisions: Divisions = divisionsNow("countries")

    let dirty = true
    const invalidate = () => {
        dirty = true
    }

    const slots = new FlagSlots(invalidate)

    // The flags painted in each landmass's frame, and in each province's.
    const territories = new BorderField(borders, undefined, leastPaintedTiles(scale))
    const provinceFrames = new BorderField({frames: provinces.frames, totals: provinces.tiles}, undefined, leastPaintedTiles(scale))
    const framesOf = (by: Divisions) => by === "provinces" ? provinceFrames : territories
    const spans = new ProvinceSpans(provinces, borders.codes.length)

    const uniforms: Uniforms = {
        pointSize: {value: tilePointSize(camera.zoom, layoutViewport().height, scale) * renderer.getPixelRatio()},
        virusLift: {value: virusLiftOf(scale)},
        leastFlagRadius: {value: leastFlagRadiusOf(scale)},
        atlasTexture: {value: textureLoader.load(ATLAS_URL)},
        atlasTextureSize: {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)},
        landmassData: {value: territories.texture},
        landmassCount: {value: territories.count},
        provinceData: {value: provinceFrames.texture},
        provinceCount: {value: provinceFrames.count},
        provinceShare: {value: divisions === "provinces" ? 1 : 0},
        pixelsPerRadian: {value: 1},
        pixelRatio: {value: renderer.getPixelRatio()},
        flagPaint: {value: flagPaint(camera.zoom, layoutViewport().height, mapView, scale)},
        splitFlags: {value: splitFlagsOf(mapView)},
        shieldMost: {value: SHIELD_MOST_UNTIL_READ},
        shieldMarks: {value: drawShieldMarks(SHIELD_MOST_UNTIL_READ, invalidate)},
        shieldCells: {value: shieldCellsOf(SHIELD_MOST_UNTIL_READ)},
        highlight: {value: new THREE.Vector2(NO_PROVINCES.first, NO_PROVINCES.last)},
        slotTexture: slots.texture,
        slotFrames: slots.frames,
        pulseTime: {value: 0},
        ...blastUniforms(prefersReducedMotion()),
        ...fortifyUniforms(),
    };

    const field = new TileField(uniforms, geometryData, graphics);

    const landmasses = new Landmasses(borders.assignment, borders.codes.length)
    const provinceName = (province: number) => {
        const landmass = provinces.landmass[province] ?? 0
        return translator().province(province, landmass, borders.codes[landmass])
    }
    field.setProvinces(provinces.province)

    // The tiles of each province, and how much of each is on screen, made the first time provinces divide the map.
    let provinceTiles: Landmasses | undefined
    const tilesOfProvince = (province: number) => (provinceTiles ??= new Landmasses(provinces.province, provinces.landmass.length)).tilesOf(province)
    const tileArea = tileAreaOf(board.lattice.cols)
    const sight = new LandmassSight((landmass) => landmasses.tilesOf(landmass), borders.codes.length, geometryData.positions, tileArea)
    let provinceSight: LandmassSight | undefined
    const sightOf = (by: Divisions) => by === "countries"
        ? sight
        : provinceSight ??= new LandmassSight(tilesOfProvince, provinces.landmass.length, geometryData.positions, tileArea)
    const settling = new Settling()
    // The view the flags were last ranked for: a recount ranks them for it too, never for a camera in motion.
    let settled: {view: ScreenView, areas: Float32Array} | undefined
    const settle = () => {
        const view = screenViewOf(camera, renderer.domElement)
        settled = {view, areas: sightOf(divisions).areasOnScreen(view)}
        return settled
    }
    const viewSize = new THREE.Vector2()

    const arrangeSlots = () => {
        if (flagPaint(camera.zoom, renderer.getSize(viewSize).y, mapView, scale) <= 0) return
        const {view, areas} = settled ?? settle()
        slots.arrange(framesOf(divisions).flagNeeds(view, areas), slotBudgetOf(renderer.domElement.width, renderer.domElement.height))
    }

    const sphere = innerSphere()
    const picker = new TilePicker(renderer.domElement, geometryData.cells);
    const territoryLayer = createTerritoryLayer({
        sphere,
        cells: geometryData.cells,
        terrain,
        tileData: field.tileData,
        provinceLandmass: provinces.landmass,
        flagRectangles: field.flagRectangles,
        uniforms,
    })
    scene.add(territoryLayer)
    const clock = new MapClock()
    const tiles = new MapTiles(field.size, TILES_PER_BATCH, clock)
    const ownership = tiles.ownership
    const shielded = tiles.shields
    const holdings = new Holdings()

    // Kept painted over the reads and the server's counts: a reload shows them until they run out.
    const ghosts = new Ghosts(savedGhostTakes(field.size), Date.now())
    field.setOwners(ghosts.snapshot().map(({tile, flag}) => ({tile, country: flag})))
    const shownOwnerOf = (tile: number) => ghosts.flagOf(tile) ?? ownership.ownerOf(tile)

    let country: Country = initialCountry;
    let crowd: number | undefined
    let inBattle = false

    const bonusBox = createBonusBox()
    scene.add(bonusBox.object)

    const bonusPointer = createBonusPointer(eventTarget)

    const enclosures = createEnclosureEffects(geometryData.positions, scale)
    scene.add(enclosures.object)

    const bonusClicks = createClickEffects(geometryData.positions, scale)
    scene.add(bonusClicks.object)

    const plainClicks = createClickGlints(geometryData.positions, scale)
    scene.add(plainClicks.object)

    const outline = createBorderLines(borderLines, scale)
    scene.add(outline.object)
    // Read once the globe is up: nothing is drawn with them in the landing view of the countries' divisions.
    const {provinceLines} = board.urls
    if (provinceLines && board.files.provinces) {
        loadProvinceLines(provinceLines, coarseTolerancesOf(scale), lifetime.signal).then((read) => {
            if (lifetime.signal.aborted) return
            if (read.countryLines !== fileHashOf(board.page.borderLines) || read.provinces !== fileHashOf(board.files.provinces!)) {
                throw new Error(`${provinceLines} was traced on other countries' lines or provinces`)
            }
            outline.takeProvinces(read)
            invalidate()
        }).catch((e) => {
            if (!lifetime.signal.aborted) console.error("could not read the province lines", e)
        })
    }

    let offered: BoxOffer | undefined

    const ownClicks = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownHits = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownPlacements = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)
    const ownEnclosures = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)

    const driveBonusBox = (seconds: number) => {
        const enclosing = enclosures.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())
        const spreading = bonusClicks.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())
        const boxed = bonusBox.update(seconds, camera)
        bonusPointer.update(bonusBox.flying ? bonusBox.object.position : undefined, camera)
        return enclosing || spreading || boxed
    }

    const driveGlints = (seconds: number) =>
        plainClicks.update(seconds, camera, renderer.domElement.height, renderer.getPixelRatio())

    const targets = new FortifyTargets({
        sizeOf: (province) => provinces.tiles[province] ?? 0,
        leaderOf: (province) => holdings.leaderOfProvince(province),
        ownerOf: (tile) => ownership.ownerOf(tile),
        known: (tile) => tiles.known(tile),
        tilesOf: tilesOfProvince,
        provinceOf: (tile) => provinces.province[tile - 1] ?? 0,
    }, (tiles, on) => {
        field.setPulses(tiles, on)
        invalidate()
    })
    // Off until the locks are read: without them a pulse would show for the flag that fortified last.
    let tracking = false

    const readFortresses = async () => {
        try {
            const locks = await ownershipsGetter.getFortresses(lifetime.signal)
            if (lifetime.signal.aborted) return
            targets.lockAll(locks, provinces.landmass.length)
            tracking = true
        } catch (e) {
            if (!lifetime.signal.aborted) console.error("could not read the fortresses", e)
        }
    }

    let front: Front | undefined

    const lightFront = ({on, off}: FrontChanges) => {
        if (on.length === 0 && off.length === 0) return
        field.setFront(off, false)
        field.setFront(on, true)
        invalidate()
    }

    const missionGround: MissionGround = {
        ownerOf: (tile) => ownership.ownerOf(tile),
        shieldsOf: (tile) => shielded.shieldsOf(tile),
        groundOf: (tile) => borders.codes[borders.assignment[tile - 1]] || undefined,
    }
    const missionMarks = new MissionMarks(missionGround, field.size)

    const markMissions = ({on, off}: FrontChanges) => {
        if (on.length === 0 && off.length === 0) return
        field.setMissionMarks(off, false)
        field.setMissionMarks(on, true)
        invalidate()
    }

    const pulsing = (camera: THREE.Camera) => {
        if (tiles.infections.count() > 0) return true
        for (const province of targets.provinces()) {
            const at = province * 5
            const along = camera.position.x * provinces.frames[at] + camera.position.y * provinces.frames[at + 1] + camera.position.z * provinces.frames[at + 2]
            if (along > 0) return true
        }
        return front?.facing(camera.position) ?? false
    }

    const publishBoard = (live: boolean) => {
        updateLeaderboard(rankCountries(ghosts.countsOver(holdings.counts(), (tile) => ownership.ownerOf(tile))), live)
    }

    const repaint = (tiles: readonly number[]) => {
        field.setOwners(tiles.map((tile) => ({tile, country: shownOwnerOf(tile)})))
        if (front) lightFront(front.touch(tiles))
        invalidate()
    }

    // The flag's last tile taken by someone else: the under-attack moment flies there.
    let lost: number | undefined

    const ghostTook = (take: GhostTake) => {
        if (shownOwnerOf(take.tile) === country.code) lost = take.tile
        ghosts.take(take)
        saveGhostTakes(ghosts.snapshot())
        repaint([take.tile])
        plainClicks.playClick(take.tile, camera)
        publishBoard(true)
    }

    const releaseGhosts = (tiles: Iterable<number>) => {
        const freed = ghosts.release(tiles)
        if (freed.length === 0) return
        saveGhostTakes(ghosts.snapshot())
        repaint(freed)
        publishBoard(true)
    }

    // A read paints under a ghost; anything else that changes a tile is the server's, and wins over it.
    const applyChanges = (changes: OwnerChange[], live = true) => {
        if (changes.length === 0) return
        const tiles = changes.map(({tile}) => tile)
        if (live) releaseGhosts(tiles)
        field.setOwners(ghosts.size === 0 ? changes : changes.filter(({tile}) => ghosts.flagOf(tile) === undefined))
        if (front) lightFront(front.touch(tiles))
        markMissions(missionMarks.touch(tiles))
        if (capitalMarks) markCapitals(capitalMarks.touch(tiles))
        if (tracking) targets.touch(tiles)
        invalidate()
    }

    // The badges count up to the most a tile holds on the map: `tileShields`, or what a seed laid past it, as a capital's
    // 100 in an arena. The sheet grows to it, by tens, and falls back to `tileShields` once no tile holds more: a hit
    // redraws it only when it crosses one of those.
    const drawMarks = (most: number) => {
        if (most === uniforms.shieldMost.value) return
        uniforms.shieldMost.value = most
        uniforms.shieldMarks.value.dispose()
        uniforms.shieldMarks.value = drawShieldMarks(most, invalidate)
        uniforms.shieldCells.value = shieldCellsOf(most)
    }
    const marksFor = (held: number) => held <= tileShieldsMost() ? tileShieldsMost() : Math.ceil(held / 10) * 10
    const fitMarks = (changes: readonly ShieldChange[]) => {
        const floor = tileShieldsMost()
        let highest = 0
        let fell = false
        for (const {shields, was} of changes) {
            highest = Math.max(highest, shields)
            if (was > floor && shields <= floor) fell = true
        }
        if (highest > uniforms.shieldMost.value) drawMarks(marksFor(highest))
        else if (fell && uniforms.shieldMost.value > floor) drawMarks(marksFor(shielded.most()))
    }

    const showShields = (changes: ShieldChange[]) => {
        if (changes.length === 0) return
        field.setShields(changes)
        fitMarks(changes)
        markMissions(missionMarks.touch(changes.map(({tile}) => tile)))
        invalidate()
    }

    const noChanges = noTileChanges

    const markInfections = (changes: readonly InfectionChange[]) => {
        if (changes.length === 0) return
        const {on, off} = infectionMarksOf(changes)
        field.setInfectedMarks(off, false)
        field.setInfectedMarks(on, true)
        invalidate()
    }

    const markMines = (changes: readonly MineChange[]) => {
        if (changes.length === 0) return
        const {on, off} = mineMarksOf(changes)
        field.setMineMarks(off, false)
        field.setMineMarks(on, true)
        invalidate()
    }

    const recount = ({countries, landmasses, provinces: moved}: Recount, live: boolean) => {
        if (landmasses.length > 0) territories.setRows(landmasses, (landmass) => holdings.leaderOf(landmass))
        if (moved.length > 0) {
            provinceFrames.setRows(moved, (province) => holdings.leaderOfProvince(province))
            if (tracking) targets.recount(moved)
        }
        if (landmasses.length > 0 || moved.length > 0) {
            arrangeSlots()
            invalidate()
        }
        if (countries) publishBoard(live)
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

    // Every rally on, of every flag; the page's flag's rings in the rally's colour, with an arrow to the newest.
    const rallies = new Rallies()
    const rallyMarks = createRallyMarks(eventTarget, rallyCallers, () => rules)
    scene.add(rallyMarks.object)
    // In CSS pixels, read off the canvas, so a frame lays nothing out.
    const canvasSize = () => ({
        width: renderer.domElement.width / renderer.getPixelRatio(),
        height: renderer.domElement.height / renderer.getPixelRatio(),
    })
    const rallyPops = createRallyPops(eventTarget)
    const pointAt = (tile: number) => {
        const at = (tile - 1) * 3
        return new THREE.Vector3(geometryData.positions[at], geometryData.positions[at + 1], geometryData.positions[at + 2])
    }

    const showRallies = () => {
        rallyMarks.show(rallies.on(monotonicNow()), country.code)
        invalidate()
    }

    let charges: Charges = NO_CHARGES

    let switches: Switches = ALL_OFF

    let rules: BonusRules | undefined

    let armed: {radius: number} | undefined

    let virusAimed = false
    let mineAimed = false
    let rallyAimed = false

    let restSwitchedOff = false

    const enclosuresUnlimited = () => rules?.enclosuresUnlimited ?? false

    const switchTo = (next: Switches) => {
        const settled = atRest(next, restSwitchedOff || armed || virusAimed || mineAimed || rallyAimed ? undefined : restingSwitch(rules))
        if (sameSwitches(settled, switches)) return
        switches = settled
        onSwitchesChange(settled)
    }

    const hold = new HoldToDrop<THREE.Vector3>({holdSeconds: HOLD_TO_DROP_SECONDS, tolerancePx: HOLD_TOLERANCE_PX})

    let swallowClick = false

    const press = new ClickOrDrag(CLICK_TOLERANCE)

    const peek = new TilePeek(PEEK_RULES)
    const territoryPeek = new TilePeek(PEEK_RULES)
    let mouse = false
    let pointerAt: Point = {x: 0, y: 0}
    let peeking: number | undefined
    // The territory under the pointer, and the one peeked: a landmass's id, or a province's while provinces divide the map.
    let peekingTerritory: number | undefined
    let hoveredTerritory: number | undefined
    let shownTerritory: Place | undefined

    const landmassOf = (tile: number | undefined) => (tile === undefined ? 0 : borders.assignment[tile - 1]) || undefined
    // The province under the pointer in orbit: a landmass's card's fortify is that province's.
    let pointedProvince: number | undefined
    const notePointed = (tile: number | undefined) => {
        if (tile !== undefined) pointedProvince = provinces.province[tile - 1] || undefined
    }
    const fortifiedIn = (landmass: number) => pointedProvince !== undefined && provinces.landmass[pointedProvince] === landmass
        ? pointedProvince
        : provinces.province[(landmasses.tilesOf(landmass)[0] ?? 0) - 1] ?? 0
    const territoryAt = (tile: number | undefined) =>
        (tile === undefined ? 0 : (divisions === "provinces" ? provinces.province : borders.assignment)[tile - 1]) || undefined
    const placeOf = (territory: number): Place => divisions === "provinces"
        ? {kind: "province", province: territory, landmass: provinces.landmass[territory]}
        : {kind: "landmass", landmass: territory}
    const tilesOfPlace = (place: Place) => place.kind === "landmass" ? landmasses.tilesOf(place.landmass) : tilesOfProvince(place.province)

    const orbiting = () => inOrbit(camera.zoom, renderer.domElement.height / renderer.getPixelRatio(), scale)

    const peekAt = ({spot: tile, point, touch}: Peeked) => {
        const owner = ownership.ownerOf(tile)
        if (!owner || ghosts.flagOf(tile) !== undefined) return
        peeking = tile
        onPeek({tile, countryCode: owner, ground: missionGround.groundOf(tile), shields: shielded.shieldsOf(tile), point, touch})
    }

    const endPeek = () => {
        if (peeking === undefined) return
        peeking = undefined
        onPeekEnd()
    }

    const peekTerritoryAt = ({spot: territory, point, touch}: Peeked) => {
        peekingTerritory = territory
        onTerritoryPeek({place: placeOf(territory), point, touch})
    }

    const endTerritoryPeek = () => {
        if (peekingTerritory === undefined) return
        peekingTerritory = undefined
        onTerritoryPeekEnd()
    }

    let flight: Flight | undefined
    let taken = false

    const territoryShot = (tiles: ArrayLike<number>) => {
        const {width, height} = renderer.domElement
        const {least, most} = flownZoomsOf(height / renderer.getPixelRatio(), scale)
        const points = Array.from(tiles, (tile) => pointOf(geometryData.positions, tile))
        return territoryShotOf(points, width / height, least, Math.min(most, maxZoom))
    }

    const fly = (to: Shot) => {
        const {x, y, z} = camera.position.clone().normalize()
        flight = {
            from: {direction: {x, y, z}, zoom: camera.zoom},
            to,
            since: performance.now() / 1000,
            seconds: prefersReducedMotion() ? 0 : FLIGHT_SECONDS,
        }
    }

    const flyTo = (territory: number | undefined) => {
        if (territory === undefined) return
        const to = territoryShot(tilesOfPlace(placeOf(territory)))
        if (to) fly(to)
    }

    const flyDown = (tile: number) => {
        const {most} = flownZoomsOf(renderer.domElement.height / renderer.getPixelRatio(), scale)
        fly({direction: pointOf(geometryData.positions, tile), zoom: Math.min(most, maxZoom)})
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
            taken = true
        },
    }

    // The flag's own country is its biggest landmass.
    const homeOf = (code: string) => {
        let home: number | undefined
        for (let landmass = 1; landmass < borders.codes.length; landmass++) {
            if (borders.codes[landmass] !== code) continue
            if (home === undefined || landmasses.tilesOf(landmass).length > landmasses.tilesOf(home).length) home = landmass
        }
        return home
    }

    const home = showFront && !linkedShot ? homeOf(country.code) : undefined
    const homeShot = home === undefined ? undefined : territoryShot(landmasses.tilesOf(home))
    if (homeShot) aim(camera, homeShot)

    let army: GhostArmy | undefined

    let neighbours: Neighbours | undefined
    const neighbourhood = () => neighbours ??= new Neighbours(geometryData.positions, scale)

    const ghostGround = (neighbours: Neighbours): Ground => ({
        positions: geometryData.positions,
        neighboursOf: (tile) => neighbours.of(tile),
        ownerOf: (tile) => ownership.ownerOf(tile),
        shieldsOf: (tile) => shielded.shieldsOf(tile),
        groundOf: (tile) => {
            const code = borders.codes[borders.assignment[tile - 1]]
            return code && regions.get(code) ? code : undefined
        },
        countOf: (flag) => holdings.tilesOf(flag),
    })

    const openGround: OpenGround = {
        get flag() {
            return country.code
        },
        positions: geometryData.positions,
        ownerOf: (tile) => ownership.ownerOf(tile),
        shieldsOf: (tile) => shielded.shieldsOf(tile),
        get reach() {
            return reach !== undefined
        },
    }
    const opens = opensOn(openGround)

    const tapPointer = showTap ? createTapPointer(eventTarget) : undefined
    const tapAt = new THREE.Vector3()
    let tapping = false
    let tapped: number | undefined
    let aimedAt = -Infinity

    const aimTap = (from: GlobePoint) => {
        tapped = front ? openTileNear(front, from, openGround) : undefined
    }

    const goToFront = () => {
        if (!front) {
            front = new Front(neighbourhood(), shownOwnerOf, geometryData.positions)
            lightFront(front.follow(country.code))
        }
        army = new GhostArmy(ghostGround(neighbourhood()), ghosts, Math.random, Date.now())
        tapping = showTap
        if (taken || !homeShot) return
        const open = openTileNear(front, homeShot.direction, openGround)
        const target = open ?? front.nearestTo(homeShot.direction)
        if (target !== undefined) flyDown(target)
        tapped = open
        aimedAt = performance.now() / 1000
    }

    const driveTap = (seconds: number) => {
        if (!tapPointer || !tapping) return
        if (flight || armed || budget?.frozen || orbiting()) {
            tapPointer.update(undefined, camera)
            return
        }
        if (tapped !== undefined && !opens(tapped)) tapped = undefined
        if (tapped === undefined && seconds >= aimedAt + TAP_AIM_EVERY_SECONDS) {
            aimedAt = seconds
            const {x, y, z} = camera.position.clone().normalize()
            aimTap({x, y, z})
        }
        if (tapped === undefined) {
            tapPointer.update(undefined, camera)
            return
        }
        const at = (tapped - 1) * 3
        tapPointer.update(tapAt.set(geometryData.positions[at], geometryData.positions[at + 1], geometryData.positions[at + 2]), camera)
    }

    const endTap = () => {
        tapping = false
        tapped = undefined
        tapPointer?.update(undefined, camera)
    }

    // The reach and the capitals rules, each followed as the rules in force say: the reach draws the bridges and rings
    // what the flag can take, across them too; the capitals rules mark each capital still standing and keep shields off
    // a fixed one.
    let capitalsFile: Promise<Capitals> | undefined
    let bridgesFile: Promise<Bridges> | undefined
    let arcs: BridgeArcs | undefined
    let capitals: Capitals | undefined
    let capitalMarks: CapitalMarks | undefined
    let reach: Front | undefined

    const capitalRules = (read: BonusRules | undefined) => (read?.knockOut || read?.capitalShieldsFixed) ?? false
    const tileShieldsMost = () => rules && rules.tileShields > 0 ? rules.tileShields : SHIELD_MOST_UNTIL_READ
    const fixedCapitals = () => rules?.capitalShieldsFixed ? capitals : undefined

    const publishStanding = () => {
        onStanding(rules?.knockOut && capitals ? standingOf(capitals, tiles.knockedOut) : undefined)
    }

    const markCapitals = ({on, off}: FrontChanges) => {
        if (on.length === 0 && off.length === 0) return
        field.setCapitalMarks(off, false)
        field.setCapitalMarks(on, true)
        invalidate()
    }

    const playReach = (bridges: Bridges) => {
        if (front) lightFront(front.drop())
        reach = new Front(acrossBridges(neighbourhood(), bridges), shownOwnerOf, geometryData.positions)
        front = reach
        lightFront(reach.follow(country.code))
        arcs = createBridgeArcs(bridges.all, geometryData.positions)
        scene.add(arcs.object)
        invalidate()
    }

    const dropArcs = () => {
        if (!arcs) return
        scene.remove(arcs.object)
        arcs.dispose()
        arcs = undefined
        invalidate()
    }

    const leaveReach = () => {
        if (!reach) return
        lightFront(reach.drop())
        dropArcs()
        reach = undefined
        front = showFront && loaded ? new Front(neighbourhood(), shownOwnerOf, geometryData.positions) : undefined
        if (front) lightFront(front.follow(country.code))
    }

    const followReach = (on: boolean) => {
        if (!on) {
            leaveReach()
            return
        }
        bridgesFile ??= loadBridges(board.urls.bridges, field.size, lifetime.signal)
        bridgesFile.then((bridges) => {
            if (rules?.reach && !reach && !lifetime.signal.aborted) playReach(bridges)
        }, (e) => {
            bridgesFile = undefined
            if (!lifetime.signal.aborted) console.error("could not read the map's bridges", e)
        })
    }

    const playCapitals = (read: Capitals) => {
        capitals = read
        capitalMarks = new CapitalMarks(read, (tile) => ownership.ownerOf(tile))
        markCapitals(capitalMarks.follow())
        publishStanding()
    }

    const leaveCapitals = () => {
        if (capitalMarks) markCapitals(capitalMarks.drop())
        capitalMarks = undefined
        capitals = undefined
        publishStanding()
    }

    const followCapitals = (on: boolean) => {
        if (!on) {
            if (capitals) leaveCapitals()
            return
        }
        if (capitals) {
            publishStanding()
            return
        }
        capitalsFile ??= loadCapitals(board.urls.capitals, field.size, lifetime.signal)
        capitalsFile.then((read) => {
            if (capitalRules(rules) && !capitals && !lifetime.signal.aborted) playCapitals(read)
        }, (e) => {
            capitalsFile = undefined
            if (!lifetime.signal.aborted) console.error("could not read the map's capitals", e)
        })
    }

    const followRules = (read: BonusRules) => {
        followReach(read.reach)
        followCapitals(capitalRules(read))
    }

    // The territory under the pointer, else the one whose card is open, lit in orbit: its tiles and its outline.
    let lit = ""
    const highlight = (orbit: boolean) => {
        const place = !orbit ? undefined : hoveredTerritory !== undefined ? placeOf(hoveredTerritory) : shownTerritory
        const key = place ? placeKey(place) : ""
        if (key === lit) return false
        lit = key
        const span = place === undefined ? NO_PROVINCES : place.kind === "province" ? {first: place.province, last: place.province} : spans.of(place.landmass)
        uniforms.highlight.value.set(span.first, span.last)
        if (place?.kind === "province") outline.setHighlight("provinces", place.province)
        else outline.setHighlight("countries", place?.landmass ?? 0)
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

    const putAway = () => {
        disarm()
        switchTo(switches)
    }

    const stopAimingVirus = () => {
        if (!virusAimed) return
        virusAimed = false
        eventTarget.classList.remove("viewer-canvas--infecting")
        onVirusAimChange(false)
    }

    const putVirusAway = () => {
        stopAimingVirus()
        switchTo(switches)
    }

    const stopAimingMine = () => {
        if (!mineAimed) return
        mineAimed = false
        eventTarget.classList.remove("viewer-canvas--mining")
        onMineAimChange(false)
    }

    const putMineAway = () => {
        stopAimingMine()
        switchTo(switches)
    }

    const stopAimingRally = () => {
        if (!rallyAimed) return
        rallyAimed = false
        eventTarget.classList.remove("viewer-canvas--rallying")
        onRallyAimChange(false)
    }

    const putRallyAway = () => {
        stopAimingRally()
        switchTo(switches)
    }

    const aimRally = () => {
        if (rallyAimed || charges.rallies === 0 || !rallier || rules?.ralliesOff) return
        disarm()
        stopAimingVirus()
        stopAimingMine()
        rallyAimed = true
        eventTarget.classList.add("viewer-canvas--rallying")
        onRallyAimChange(true)
        switchTo(ALL_OFF)
    }

    const aimMine = () => {
        if (mineAimed || charges.mines === 0 || !miner) return
        disarm()
        stopAimingVirus()
        stopAimingRally()
        mineAimed = true
        eventTarget.classList.add("viewer-canvas--mining")
        onMineAimChange(true)
        switchTo(ALL_OFF)
    }

    const aimVirus = () => {
        if (virusAimed || charges.viruses === 0 || !infector) return
        disarm()
        stopAimingMine()
        stopAimingRally()
        virusAimed = true
        eventTarget.classList.add("viewer-canvas--infecting")
        onVirusAimChange(true)
        switchTo(ALL_OFF)
    }

    const arm = () => {
        if (armed || charges.bombs === 0 || !bomber || !rules || rules.bombsOff) return
        stopAimingVirus()
        stopAimingMine()
        stopAimingRally()
        armed = {radius: rules.blastRadius}
        eventTarget.classList.add("viewer-canvas--armed")
        onArmedChange(true)
        switchTo(ALL_OFF)
    }

    const takeCharges = (held: Charges) => {
        charges = held
        if (held.bombs === 0) disarm()
        if (held.viruses === 0) stopAimingVirus()
        if (held.mines === 0) stopAimingMine()
        if (held.rallies === 0) stopAimingRally()
        switchTo(switchesHeld(switches, held, rules))
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
    const stopBoxes = boxCatcher?.listenForBoxes((offer) => {
        offered = offer
        bonusBox.spawn(offer.seed, (offer.expiresAt - CLAIM_MARGIN_MS) / 1000)
        playSound("bonusSpawn")
    })
    const stopBonuses = bonusListener?.listenForBonuses({
        onEnclosed: (enclosure) => {
            tiles.heard(enclosure.version)
            const struck = new Set(enclosure.struck.map(({tile}) => tile))
            const taken = enclosure.filled.filter((tile) => !struck.has(tile))
            releaseGhosts(enclosure.filled)
            if (enclosure.countryId !== country.code) lost = taken.find((tile) => ownership.ownerOf(tile) === country.code) ?? lost
            const changes = noChanges()
            tiles.applyEnclosure(enclosure.countryId, taken, enclosure.struck, enclosure.version, changes)
            applyChanges(changes.owners)
            showShields(changes.shields)
            markInfections(changes.infections)
            markMines(changes.mines)
            playShields(changes.shields)
            enclosures.play(enclosure)
            if (!enclosure.yours) return
            playSound("enclose")
            ownEnclosures.record(enclosure.closingTile, enclosure.countryId, performance.now() / 1000)
        },
        onVirusSpread: (spread) => {
            releaseGhosts([...spread.taken, ...spread.struck.map(({tile}) => tile)])
            lost = spread.taken.find((tile) => ownership.ownerOf(tile) === country.code) ?? lost
            const changes = noChanges()
            tiles.applyInfection(spread, changes)
            applyChanges(changes.owners)
            showShields(changes.shields)
            playShields(changes.shields)
            markInfections(changes.infections)
            markMines(changes.mines)
        },
        onMines: (read) => {
            const changes = noChanges()
            if (read.countryId === country.code) tiles.readMines(read, changes)
            else tiles.forgetMines(changes)
            markMines(changes.mines)
        },
        onMineLaid: (laid) => {
            if (laid.countryId !== country.code) return
            const changes = noChanges()
            tiles.applyMineLaid(laid, changes)
            markMines(changes.mines)
        },
        onRallies: (read) => {
            rallies.replace(read)
            showRallies()
        },
        onRallyCalled: (rally) => {
            rallies.keep(rally)
            showRallies()
        },
        onMinesExploded: (explosion) => {
            releaseGhosts(explosion.blasts.flatMap((blast) => [...blast.taken, ...blast.struck.map(({tile}) => tile)]))
            lost = explosion.blasts.filter((blast) => blast.countryId !== country.code)
                .flatMap((blast) => blast.taken).find((tile) => ownership.ownerOf(tile) === country.code) ?? lost
            const changes = noChanges()
            tiles.applyExplosion(explosion, changes)
            applyChanges(changes.owners)
            showShields(changes.shields)
            playShields(changes.shields)
            markInfections(changes.infections)
            markMines(changes.mines)
            explode(explosion.blasts.map((blast) => blast.tile), explosion.blasts.some((blast) => blast.countryId === country.code))
        },
        onSpread: (spread) => {
            bonusClicks.playSpread(spread)
            if (ownClicks.has(spread.tile, spread.countryId, performance.now() / 1000)) playSound("spread")
        },
        onCharges: (held) => takeCharges(held),
        onRules: (read) => {
            const spreadChanged = read.spreadUnlimited !== (rules?.spreadUnlimited ?? false) ||
                read.spreadOff !== (rules?.spreadOff ?? false)
            const encloseChanged = read.enclosuresUnlimited !== enclosuresUnlimited() ||
                read.enclosuresOff !== (rules?.enclosuresOff ?? false)
            rules = read
            if (spreadChanged || encloseChanged) {
                restSwitchedOff = false
                const held = {...switches, spread: switches.spread && !spreadChanged, enclose: switches.enclose && !encloseChanged}
                switchTo(switchesHeld(held, charges, read))
            }
            drawMarks(marksFor(shielded.most()))
            followRules(read)
            if (read.bombsOff) disarm()
            if (read.ralliesOff) putRallyAway()
            invalidate()
            onRules(read)
        },
    })

    let ownDropAt: number | undefined

    const dropBomb = (point: THREE.Vector3) => {
        if (!bomber) return
        putAway()
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
        releaseGhosts([...cleared, ...struck.map(({tile}) => tile)])
        lost = cleared.find((tile) => ownership.ownerOf(tile) === country.code) ?? lost
        const changes = noChanges()
        tiles.applyBlast(cleared, struck, version, changes)
        applyChanges(changes.owners)
        showShields(changes.shields)
        markInfections(changes.infections)
        markMines(changes.mines)
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
        const landmass = landmassOf(drop.tile)
        onBombDropped(drop, landmass === undefined ? undefined : translator().landmass(landmass, borders.codes[landmass]))
    })

    // Each blast drawn as a bomb's at its impact, with no fall: the take that set it off was the fall.
    const explode = (mines: readonly number[], ours: boolean) => {
        const seconds = performance.now() / 1000
        const radius = mineBlastRadius(rules?.mineRings ?? 0, scale)
        for (const tile of mines) {
            const {x, y, z} = pointOf(geometryData.positions, tile)
            blasts.start(new THREE.Vector3(x, y, z), radius, seconds - BLAST_TIMELINE.fall)
        }
        invalidate()
        playSound("bomb", {volume: ours ? 1 : DISTANT_BOMB_VOLUME})
    }

    // Any point of the globe takes a rally, on land or sea. A rally of the caller still on is refused by the server
    // alone: the rally stays held.
    const callRally = (point: THREE.Vector3) => {
        if (!rallier) return
        putRallyAway()
        playSound("click")
        rallier.callRally({x: point.x, y: point.y, z: point.z}, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            if (e instanceof RallyRefusedError) onNotice("rallyOn")
            else if (e instanceof AllegianceLockedError) onAllegianceLocked(e.allegiance)
            else reportClaimFailure(e, {onSessionUnavailable, onBanned})
        })
    }

    // The tap decides the tile first (`layingOf`); the server checks again. The mine stays aimed while one is held, so
    // mines are laid together.
    const layMine = (tile: number) => {
        if (!miner) return
        playSound("click")
        plainClicks.playOwnClick(tile, camera)
        miner.layMine(tile, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            if (e instanceof MineRefusedError) onNotice("mineNotYours")
            else if (e instanceof AllegianceLockedError) onAllegianceLocked(e.allegiance)
            else reportClaimFailure(e, {onSessionUnavailable, onBanned})
        })
    }

    // A tile of no other flag, or with a shield, is refused here before the virus leaves the hand; the server checks again.
    const infect = (tile: number, owner: string | undefined, shields: number) => {
        if (!infector) return
        if (owner === undefined || owner === country.code || shields > 0) {
            onNotice("virusNoEnemy")
            playSound("refused")
            return
        }
        putVirusAway()
        playSound("click")
        infector.useVirus(tile, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            if (e instanceof VirusRefusedError) onNotice("virusNoEnemy")
            else if (e instanceof OutOfReachError) onOutOfReach()
            else if (e instanceof AllegianceLockedError) onAllegianceLocked(e.allegiance)
            else reportClaimFailure(e, {onSessionUnavailable, onBanned})
        })
    }

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
        if (rallyMarks.update(monotonicNow(), camera, canvasSize())) drawing = true
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
        hoveredTerritory = undefined
        endPeek()
        endTerritoryPeek()
        if (armed && !hold.holding) aimAt(undefined)
    }, listenerOptions);

    eventTarget.addEventListener('pointerdown', (event: PointerEvent) => {
        // A touch press that ends in a drag or a long press sends no click to swallow.
        if (event.isPrimary) swallowClick = false
        press.begin(event)
        endPeek()
        endTerritoryPeek()
        const pressed = !armed && event.pointerType !== "mouse" ? canvasPosition(event) : undefined
        const pressedTile = pressed && picker.pick(camera, pressed.x, pressed.y)
        const orbit = orbiting()
        peek.begin(event, orbit || !loaded ? undefined : pressedTile, performance.now() / 1000)
        if (orbit) notePointed(pressedTile || undefined)
        territoryPeek.begin(event, orbit ? territoryAt(pressedTile) : undefined, performance.now() / 1000)
        if (!loaded || !armed || !event.isTrusted || !event.isPrimary || event.button !== 0) return

        const {x, y} = canvasPosition(event)
        const point = surfacePoint(x, y)
        if (!point) return

        hold.begin(event.pointerId, event.clientX, event.clientY, performance.now() / 1000, point)
        swallowClick = true
        aimAt(point)
    }, listenerOptions);

    window.addEventListener('keydown', (event: KeyboardEvent) => {
        if (event.key !== "Escape") return
        putAway()
        putVirusAway()
        putMineAway()
        putRallyAway()
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

            boxCatcher?.claimBox(claimed.token)
                .then((reward) => {
                    if (lifetime.signal.aborted) return
                    if (reward) onBonusWon(reward)
                })
                .catch((e) => {
                    if (lifetime.signal.aborted) return
                    reportClaimFailure(e, {onSessionUnavailable, onBanned})
                })
            return
        }

        if (armed) return

        // A rally goes on the point tapped, a tile or not; from orbit, a tap flies down first.
        if (rallyAimed && loaded && !orbiting()) {
            const point = surfacePoint(x, y)
            if (point) {
                if (budget?.frozen) {
                    onFrozen()
                    playSound("refused")
                } else {
                    callRally(point)
                }
                return
            }
        }

        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return
        peek.clicked()
        territoryPeek.clicked()

        if (orbiting()) {
            flyTo(territoryAt(tile))
            return
        }

        if (!loaded) return

        if (budget?.frozen) {
            onFrozen()
            playSound("refused")
            return
        }

        if (reach && !reach.reaches(tile)) {
            onOutOfReach()
            playSound("refused")
            return
        }

        if (!regions.get(country.code)) {
            warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
            return
        }

        const owner = ownership.ownerOf(tile)
        const shields = shielded.shieldsOf(tile)
        const ghost = ghosts.flagOf(tile)

        // Half a click, or what the rules say, inside a ring of the flag: the server decides, the page counts it until then.
        const cost = clickCostOf(tile, country.code, rallies.on(monotonicNow()), geometryData.positions, rules?.rallyCost)

        if (ghost !== undefined && owner === country.code) {
            if (!canClick(budget, monotonicNow(), cost)) {
                onRateLimited()
                playSound("refused")
                return
            }
            releaseGhosts([tile])
            playSound("click")
            plainClicks.playOwnClick(tile, camera)
            return
        }

        if (virusAimed) {
            infect(tile, owner, shields)
            return
        }

        if (mineAimed && miner) {
            const laying = layingOf(owner, country.code, tiles.mines.has(tile), (rules?.knockOut && capitals?.has(tile)) ?? false)
            if (laying === "refused") {
                onNotice("mineNotYours")
                playSound("refused")
                return
            }
            if (laying === "lay") {
                layMine(tile)
                return
            }
        }

        if (switches.shield && shielder) {
            const placement = placementOf(owner, country.code, shields, rules?.tileShields, fixedCapitals()?.has(tile) ?? false)
            if (placement === "full") {
                onNotice("shieldFull")
                return
            }
            if (placement === "capital") {
                onNotice("shieldCapital")
                return
            }
            if (placement === "place") {
                placeShield(tile)
                return
            }
        }

        if (!canClick(budget, monotonicNow(), cost)) {
            onRateLimited()
            playSound("refused")
            return
        }

        if (ghost !== undefined) releaseGhosts([tile])
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
        // What the click took from the bank, which the bank shows in tenths: "-0.5", never a multiplier.
        if (cost < WHOLE_CLICK) rallyPops.pop(pointAt(tile), camera, translator().number(-cost, {maximumFractionDigits: 2}))

        const clicked = country.code
        const shielding = switches.shield && shielder !== undefined
        const mining = mineAimed && miner !== undefined
        const enclosing = switches.enclose && outcome === "taken" && !enclosuresUnlimited()
        tileClicker.clickTile(tile, clicked, switches, cost).then(() => {
            if (lifetime.signal.aborted) return
            if (outcome === "taken") endTap()
            onClickAccepted({country: clicked, took: outcome === "taken"})
            if (shielding) onNotice(outcome === "taken" ? "shieldTaken" : "shieldNotYours")
            if (mining) onNotice(outcome === "taken" ? "mineTaken" : "mineNotYours")
            if (enclosing) setTimeout(() => {
                if (lifetime.signal.aborted || ownEnclosures.has(tile, clicked, performance.now() / 1000)) return
                onNotice("nothingEnclosed")
            }, ENCLOSURE_WAIT_MS)
        }, (e) => {
            if (lifetime.signal.aborted) return
            applyChanges(ownership.rollback(claim))
            if (reportClickFailure(e, {onRateLimited, onFrozen, onVPNBlocked, onBanned, onSessionUnavailable, onAllegianceLocked, onOutOfReach})) playSound("refused")
        })
    }, listenerOptions);

    const resizeListener = () => {
        const {width, height} = layoutViewport();

        camera.left = -cameraSize * (width / height);
        camera.right = cameraSize * (width / height);
        camera.updateProjectionMatrix();

        renderer.setPixelRatio(sharpness.ratio());
        renderer.setSize(width, height);
        invalidate();
    };
    window.addEventListener('resize', resizeListener, listenerOptions);

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch((updates: Update[]) => {
        releaseGhosts(updates.map(({tile}) => tile))
        const changes = noChanges()
        tiles.applyUpdates(updates, changes)
        applyChanges(changes.owners)
        showShields(changes.shields)
        markInfections(changes.infections)
        markMines(changes.mines)
        playShields(changes.shields)

        const seconds = performance.now() / 1000
        for (const {tile, clicked, previousCountry, newCountry} of updates) {
            if (previousCountry === country.code && newCountry !== country.code) lost = tile
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
        markInfections(changes.infections)
        markMines(changes.mines)
        if (capitals) publishStanding()
        if (tracking) targets.touch(rangeTiles(batch.range))
        if (!loaded) {
            const share = tiles.readShare()
            if (share < 1) onLoadProgress(share)
            if (share === 1) {
                loaded = true
                drawingWhole = true
                invalidate()
                tilesWereRead()
                sharpness.startMeasuring(performance.now())
                releaseGhosts(ghosts.snapshot().filter(({tile}) => ownership.ownerOf(tile) === undefined).map(({tile}) => tile))
                if (showFront) goToFront()
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
        const {province} = fortification
        const members = tilesOfProvince(province)
        releaseGhosts(members)
        const fixed = fixedCapitals()
        const shielding = fixed ? members.filter((tile) => !fixed.has(tile)) : members
        const raised = tiles.applyFortification(shielding, fortification.countryId, tileShieldsMost(), fortification.version)
        markMissions(missionMarks.touch(raised.map(({tile}) => tile)))
        if (document.hidden || uniforms.motion.value === 0) showShields(raised)
        else waves.start({first: province, last: province}, members, fortification.tile, performance.now() / 1000)

        const own = fortification.countryId === country.code
        const news = isNews(fortification, members.length, country.code, scale)
        if (news) playSound("fortify", {volume: own ? 1 : DISTANT_BOMB_VOLUME})
        onFortified({fortification, name: provinceName(province), tiles: members.length, news})
        if (tracking) targets.lock(province, fortification.countryId)
    })

    const stopKnockOuts = updatesListener.listenForKnockOuts((knockOut) => {
        const changes = noChanges()
        tiles.applyKnockOut(knockOut.countryId, knockOut.cleared, knockOut.version, changes)
        applyChanges(changes.owners)
        showShields(changes.shields)
        markInfections(changes.infections)
        markMines(changes.mines)
        publishStanding()
    })

    const atmosphere = addDisplayObjects(scene, field.displayPoints, graphics, sphere)
    // A bomb's effects built and first drawn at its drop froze the page for ~0.8 s: both happen at load.
    void renderer.compileAsync(scene, camera).then(() => blasts.prime())

    let captureRequests: CaptureRequest[] = []

    const takeCaptureRequests = () => {
        const waiting = captureRequests
        captureRequests = []
        return waiting
    }

    const ghostSight = (): Sight | undefined => {
        const {width, height} = renderer.domElement
        const centre = surfacePoint(width / 2, height / 2)?.normalize()
        if (!centre) return undefined
        const half = Math.min(camera.right - camera.left, camera.top - camera.bottom) / 2 / camera.zoom
        const reach = Math.min(1, GHOSTS_SIGHT * half)
        return {flag: country.code, centre: {x: centre.x, y: centre.y, z: centre.z}, least: Math.sqrt(1 - reach * reach)}
    }

    let ghostsAt = -Infinity

    const driveGhosts = (seconds: number) => {
        if (seconds < ghostsAt + GHOSTS_EVERY_SECONDS || budget?.frozen) return
        ghostsAt = seconds
        const now = Date.now()
        const expired = ghosts.expire(now)
        if (expired.length > 0) {
            saveGhostTakes(ghosts.snapshot())
            repaint(expired)
            for (const tile of expired) plainClicks.playClick(tile, camera)
            publishBoard(true)
        }

        const virusAtWork = virusAimed || tiles.infections.count() > 0
        if (virusAtWork) releaseGhosts(ghosts.snapshot().map(({tile}) => tile))
        if (!army || inBattle || virusAtWork || crowd === undefined || crowd >= GHOSTS_BELOW || orbiting()) return
        const sight = ghostSight()
        const take = sight && army.march(now, sight)
        if (take && !tiles.mines.has(take.tile)) ghostTook(take)
    }

    // The zoom crossed the provinces' line: the flags fade to the other frames, and the pointer finds its territory by
    // the other divisions.
    const divide = (next: Divisions) => {
        divisions = next
        territoryPeek.hover(undefined, 0, pointerAt)
        endTerritoryPeek()
        hoveredTerritory = undefined
        pendingPointer ??= mouseOnCanvas
        // The camera is in motion: the flags are ranked again for the new frames once it comes to rest.
        settled = undefined
        settling.rankAgain()
        invalidate()
    }

    const reducedMotion = prefersReducedMotion()
    let fadedAt: number | undefined
    // A step of the fade to the divisions drawn, each frame until it is done: true while it runs.
    const fade = (seconds: number): boolean => {
        const share = uniforms.provinceShare.value
        if (share === (divisions === "provinces" ? 1 : 0)) {
            fadedAt = undefined
            return false
        }
        uniforms.provinceShare.value = provinceShareAfter(share, divisions, fadedAt === undefined ? 0 : seconds - fadedAt, reducedMotion)
        fadedAt = seconds
        invalidate()
        return true
    }

    const {stop: stopAnimation} = startAnimation(renderer, scene, camera, uniforms, scale, director, pilot, () => mapView, () => {
        const was = dirty
        dirty = false
        return was
    }, (seconds) => {
        sharpness.tick(seconds * 1000)
        const motion = settling.tick(seconds * 1000, viewpointOf(camera, renderer.domElement))
        if (motion === "settled") {
            settle()
            arrangeSlots()
        }
        if (motion !== "moving") slots.upload(renderer)
        driveGhosts(seconds)
        driveTap(seconds)
        const boxed = driveBonusBox(seconds)
        const fortifying = waves.update(seconds)
        const blasting = driveBlasts(seconds) || fortifying
        const shown = divisionsNow(divisions)
        if (shown !== divisions) divide(shown)
        fade(seconds)
        outline.update(camera.zoom, renderer.domElement.width, renderer.domElement.height, renderer.getPixelRatio(), mapView, uniforms.provinceShare.value)
        // The painted flags show whole or not at all: no batch fills the map on screen.
        if (loaded) field.uploadTiles(renderer)

        const peeked = armed ? undefined : peek.tick(seconds)
        if (peeked !== undefined) peekAt(peeked)
        const peekedTerritory = armed ? undefined : territoryPeek.tick(seconds)
        if (peekedTerritory !== undefined) peekTerritoryAt(peekedTerritory)

        const orbit = orbiting()
        if (orbit !== wasInOrbit) {
            wasInOrbit = orbit
            pendingPointer ??= mouseOnCanvas
        }
        const lit = highlight(orbit)

        if (pendingPointer === undefined) return boxed || blasting || lit
        const {x, y} = pendingPointer
        pendingPointer = undefined
        if (armed) {
            field.setHover(undefined)
            hoveredTerritory = undefined
            highlight(orbit)
            if (!hold.holding) aimAt(surfacePoint(x, y))
            return true
        }
        const hovered = picker.pick(camera, x, y)
        if (mouse) {
            const tile = orbit ? undefined : hovered
            const territory = orbit ? territoryAt(hovered) : undefined
            if (orbit) notePointed(hovered)
            peek.hover(tile, seconds, pointerAt)
            territoryPeek.hover(territory, seconds, pointerAt)
            if (tile !== peeking) endPeek()
            if (territory !== peekingTerritory) endTerritoryPeek()
            hoveredTerritory = territory
        }
        const relit = highlight(orbit)
        return field.setHover(hovered) || relit || lit || boxed || blasting
    }, () => {
        field.cull(camera)
        outline.cull(camera)
    }, () => {
        undoShake()
        // Out of this frame: sizing the canvas again clears it, and this frame is drawn.
        if (sharpness.drew()) setTimeout(resizeListener)
        if (drawingWhole) {
            drawingWhole = false
            onLoadProgress(1)
        }
        if (captureRequests.length === 0) return

        const frame = readDrawingBuffer(renderer)
        for (const request of takeCaptureRequests()) request.resolve(frame)
    }, pulsing, driveGlints);

    const globe: Globe = {
        tilesCount: field.size,
        tilesRead,
        setCountry: (newCountry: Country) => {
            const changed = newCountry.code !== country.code
            country = newCountry
            if (changed) {
                lost = undefined
                const changes = noChanges()
                tiles.forgetMines(changes)
                markMines(changes.mines)
                showRallies()
            }
            if (front && changed) lightFront(front.follow(newCountry.code))
        },
        setCrowd: (count: number | undefined) => {
            crowd = count
        },
        setBattle: (on: boolean) => {
            inBattle = on
            if (on) releaseGhosts(ghosts.snapshot().map(({tile}) => tile))
        },
        setMapView: (view: MapView) => {
            if (view === mapView) return
            mapView = view
            uniforms.splitFlags.value = splitFlagsOf(view)
            arrangeSlots()
            invalidate()
        },
        setHalo: (halo: Halo) => {
            paintHalo(atmosphere, halo)
            invalidate()
        },
        takeReward,
        setArmed: (on: boolean) => on ? arm() : putAway(),
        setVirusAimed: (on: boolean) => on ? aimVirus() : putVirusAway(),
        setMineAimed: (on: boolean) => on ? aimMine() : putMineAway(),
        setRallyAimed: (on: boolean) => on ? aimRally() : putRallyAway(),
        setSwitch: (name: keyof Switches, on: boolean) => {
            if (name === restingSwitch(rules)) restSwitchedOff = !on
            const next = switchesHeld(switched(switches, name, on), charges, rules)
            if (next[name]) {
                disarm()
                stopAimingVirus()
                stopAimingMine()
                stopAimingRally()
            }
            switchTo(next)
        },
        setClickHue: (hue: number | undefined) => plainClicks.setOwnHue(hue),
        territoryOf: (place: Place) => {
            const {landmass} = place
            const code = borders.codes[landmass]
            const own = place.kind === "landmass"
                ? {name: translator().landmass(landmass, code), size: territories.sizeOf(landmass), held: holdings.holdingsOf(landmass)}
                : {name: provinceName(place.province), size: provinceFrames.sizeOf(place.province), held: holdings.holdingsOfProvince(place.province)}
            // A province's card is about the province; a landmass's foot about the province under the pointer.
            const province = place.kind === "province" ? place.province : fortifiedIn(landmass)
            const named = place.kind === "landmass" && provinces.tiles[province] !== territories.sizeOf(landmass)
            return {
                place,
                name: own.name,
                ground: code || undefined,
                size: own.size,
                holdings: rankHoldings(own.held),
                province,
                provinceName: named ? provinceName(province) : undefined,
                fortifiedBy: targets.lockOf(province),
                target: targets.targetOf(province),
            }
        },
        showTerritory: (place: Place | undefined) => {
            shownTerritory = place
        },
        flyToLastLoss: () => {
            if (lost !== undefined) flyDown(lost)
        },
        setMissions: (targets: readonly MissionTarget[]) => {
            markMissions(missionMarks.follow(targets))
            const grounds = groundsOf(targets)
            const aimed = new Set<number>()
            for (let landmass = 1; landmass < borders.codes.length; landmass++) {
                if (grounds.has(borders.codes[landmass])) aimed.add(landmass)
            }
            outline.setAimed(aimed)
            invalidate()
        },
        flyToMission: (target: MissionTarget) => {
            const from = camera.position.clone().normalize()
            const tile = nearestPlayable(target, country.code, missionGround, geometryData.positions, from)
            if (tile === undefined) return
            pilot.cancel()
            flyDown(tile)
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
            stopKnockOuts()
            reading?.abort()
            stopBoxes?.()
            stopBonuses?.()
            stopBombs?.()
            stopWatchingBudget?.()

            bonusPointer.dispose()
            blastPointer.dispose()
            rallyMarks.dispose()
            rallyPops.dispose()
            tapPointer?.dispose()
            blasts.dispose()
            field.dispose()
            territories.dispose()
            provinceFrames.dispose()
            slots.dispose()
            outline.dispose()
            bonusBox.dispose()
            enclosures.dispose()
            bonusClicks.dispose()
            plainClicks.dispose()
            dropArcs()

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
    scale: BoardScale,
    director: Director | undefined,
    pilot: Pilot,
    mapView: () => MapView,
    takeChange: () => boolean,
    beforeRender: (seconds: number) => boolean,
    beforeDraw: () => void,
    afterRender: () => void,
    pulsing: (camera: THREE.Camera) => boolean,
    driveGlints: (seconds: number) => Glints,
): {stop: () => void} {
    const starfield = createStarfield();

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.minZoom = MIN_ZOOM;
    controls.maxZoom = maxZoomOf(scale);
    controls.panSpeed = 0.1;
    controls.enableDamping = true;
    controls.autoRotateSpeed = SPIN_TURNS_PER_MINUTE;
    controls.enabled = director === undefined;

    // A wheel zoom moves the camera inside OrbitControls' own handler, so update() misses it.
    let moved = false;

    controls.addEventListener('change', () => {
        moved = true;

        controls.rotateSpeed = (1 / camera.zoom) / 1.5;
    });

    let lastInputAt = performance.now();
    const listening = new AbortController();
    for (const input of INPUTS) {
        window.addEventListener(input, () => {
            lastInputAt = performance.now();
        }, {capture: true, passive: true, signal: listening.signal});
    }

    let interactingUntil = 0;
    controls.addEventListener('start', () => {
        interactingUntil = Infinity;
        pilot.cancel();
    });
    controls.addEventListener('end', () => {
        interactingUntil = performance.now() + INTERACTION_GRACE_MS;
    });

    let drawnAt = -Infinity;
    let beatDrawn: number | undefined;
    let tickedAt: number | undefined;
    const viewport = new THREE.Vector2();

    renderer.setAnimationLoop((time: number) => {
        const sinceLastTick = tickedAt === undefined ? 0 : time - tickedAt;
        tickedAt = time;

        controls.autoRotate = spins({zoom: camera.zoom, lastInputAt, now: time});
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
        uniforms.pointSize.value = tilePointSize(camera.zoom, height, scale) * ratio;
        uniforms.pixelRatio.value = ratio;

        uniforms.flagPaint.value = flagPaint(camera.zoom, height, mapView(), scale);
        uniforms.pixelsPerRadian.value = (renderer.domElement.height / 2) * camera.zoom;

        const moving = beforeRender(time / 1000);
        const changed = takeChange() || moving;
        const glints = driveGlints(time / 1000);

        const attended = director !== undefined || watched({lastInputAt, now: time});
        uniforms.pulseTime.value = beatClock({clock: uniforms.pulseTime.value, step: sinceLastTick / 1000, attended});
        const ambient = uniforms.pulseTime.value !== beatDrawn && pulsing(camera);
        const tick = {turned, changed, glints, ambient, attended, at: time, drawnAt, interactingUntil, sinceLastTick};
        if (!drawsFrame(tick)) return;

        moved = false;
        drawnAt = time;
        beatDrawn = uniforms.pulseTime.value;

        beforeDraw();
        starfield.render(renderer, camera, () => renderer.render(scene, camera));
        afterRender();
    });

    return {
        stop: () => {
            renderer.setAnimationLoop(null);
            listening.abort();
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

function pixelsPerRadianOf(camera: THREE.OrthographicCamera, canvas: HTMLCanvasElement): number {
    return canvas.height * camera.zoom / (camera.top - camera.bottom)
}

function viewpointOf(camera: THREE.OrthographicCamera, canvas: HTMLCanvasElement): Viewpoint {
    const {x, y, z} = camera.position
    const distance = camera.position.length()
    return {x: x / distance, y: y / distance, z: z / distance, scale: pixelsPerRadianOf(camera, canvas)}
}

function screenViewOf(camera: THREE.OrthographicCamera, canvas: HTMLCanvasElement): ScreenView {
    const pixelsPerRadian = pixelsPerRadianOf(camera, canvas)
    const right = new THREE.Vector3(1, 0, 0).applyQuaternion(camera.quaternion)
    const up = new THREE.Vector3(0, 1, 0).applyQuaternion(camera.quaternion)
    return {
        toward: new THREE.Vector3(0, 0, 1).applyQuaternion(camera.quaternion),
        right,
        up,
        origin: {x: -camera.position.dot(right) * pixelsPerRadian, y: -camera.position.dot(up) * pixelsPerRadian},
        pixelsPerRadian,
        width: canvas.width,
        height: canvas.height,
    }
}

function splitFlagsOf(view: MapView): number {
    return view === "split" ? 1 : 0
}

function prefersReducedMotion(): boolean {
    return typeof window.matchMedia === "function"
        && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

export function reportClaimFailure(
    error: unknown,
    handlers: {onSessionUnavailable: () => void, onBanned: (until: number) => void},
) {
    if (error instanceof BonusLostError || error instanceof BombRefusedError) return
    if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else if (error instanceof BannedError) handlers.onBanned(error.until)
    else console.error(error)
}

export function reportClickFailure(
    error: unknown,
    handlers: {
        onRateLimited: () => void,
        onFrozen: () => void,
        onVPNBlocked: () => void,
        onBanned: (until: number) => void,
        onSessionUnavailable: () => void,
        onAllegianceLocked: (allegiance: Allegiance) => void,
        onOutOfReach: () => void,
    },
): boolean {
    if (error instanceof RateLimitedError) handlers.onRateLimited()
    else if (error instanceof FrozenError) handlers.onFrozen()
    else if (error instanceof VPNBlockedError) handlers.onVPNBlocked()
    else if (error instanceof BannedError) handlers.onBanned(error.until)
    else if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else if (error instanceof AllegianceLockedError) handlers.onAllegianceLocked(error.allegiance)
    else if (error instanceof OutOfReachError) handlers.onOutOfReach()
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
