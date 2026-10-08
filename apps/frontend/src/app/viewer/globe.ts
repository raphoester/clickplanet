import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, pixelRatio, setupScene} from "./scene.ts";
import {graphicsOf} from "./graphics.ts";
import {loadPointGeometryData} from "./points.ts";
import {GpuPicker} from "./gpuPicking.ts";
import {CapturedFrame, readDrawingBuffer} from "./capture.ts";
import {TileField} from "./tileField.ts";
import {BorderField, countryOfTile, loadBorders} from "./borderField.ts";
import {createBorderLines, loadBorderLines} from "./borderLines.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import {BORDERS_URL} from "./bordersAsset.ts";
import {BORDER_LINES_URL} from "./borderLinesAsset.ts";
import {displayPointSize, flagPaint, tilePointSize} from "./pointSize.ts";
import {regions} from "./atlas.ts";
import {Country} from "../../domain/countries.ts";
import {MapView, Rendering} from "../../domain/displaySettings.ts";
import {
    BombDrop,
    Bomber,
    GlobePoint,
    BonusCatch,
    BonusListener,
    BonusLostError,
    BonusOffer,
    ClaimedBonus,
    ShieldRefusedError,
    OwnershipsGetter,
    RateLimitedError,
    Shielder,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "../../backends/backend.ts";
import {SessionUnavailableError} from "../../backends/session.ts";
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts";
import {OwnerChange, TileOwnership} from "../../domain/tileOwnership.ts";
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
import {now as monotonicNow} from "../../backends/clickBudget.ts";
import {BlastUniforms, blastUniforms, createBlasts} from "./blasts.ts";
import {IMPACT_DELAY} from "../../domain/blast.ts";
import {HoldToDrop} from "../../domain/holdToDrop.ts";
import {ClickOrDrag} from "../../domain/clickOrDrag.ts";
import {OwnClicks} from "../../domain/ownClicks.ts";
import {ShieldChange, outcomeOf, placementOf, TileShields} from "../../domain/shields.ts";
import {PlaySound} from "../sound/soundPlayer.ts";
import {drawShieldMarks, shieldCellsOf} from "./shieldMarks.ts";
import {AcceptedClick} from "./acceptedClicks.ts";

type Uniforms = BlastUniforms & {
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
}

const SHAKE_SECONDS = 0.5

const HOLD_TO_DROP_SECONDS = 0.7

const HOLD_TOLERANCE_PX = 6

const CLICK_TOLERANCE = {mousePx: 6, touchPx: 12}

const DISTANT_BOMB_VOLUME = 0.45

const OWN_DROP_WINDOW_SECONDS = 5

const OWN_CLICK_WINDOW_SECONDS = 3

const ENCLOSURE_WAIT_MS = 1500

const SHIELD_MOST_UNTIL_READ = 10

const TILES_PER_BATCH = 10_000

// GetMap's max-age: a map read any sooner could be older than the stream that resumed.
const CATCH_UP_DELAY_MS = 5_000

const IDLE_FRAME_MS = 16

const SPIN_TURNS_PER_MINUTE = 2

const MAX_SPIN_STEP_MS = 100

export function spinStep(sinceLastTick: number): number {
    return Math.min(Math.max(sinceLastTick, 0), MAX_SPIN_STEP_MS) / 1000
}

const INTERACTION_GRACE_MS = 1_000

export type Tick = {
    turned: boolean
    changed: boolean
    at: number
    drawnAt: number
    interactingUntil: number
    sinceLastTick: number
}

export function drawsFrame({turned, changed, at, drawnAt, interactingUntil, sinceLastTick}: Tick): boolean {
    if (!turned && !changed) return false
    if (changed || at <= interactingUntil) return true
    return at + sinceLastTick / 2 >= drawnAt + IDLE_FRAME_MS
}

const CLAIM_MARGIN_MS = 2_000

const textureLoader = new THREE.TextureLoader();

export type Shot = {
    direction: GlobePoint
    zoom: number
}

export type Director = (seconds: number) => Shot

export type GlobeOptions = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    container: HTMLElement
    country: Country
    mapView: MapView
    rendering: Rendering
    onLeaderboardChange: (entries: LeaderboardEntry[], live: boolean) => void
    onLoadProgress: (share: number) => void
    onRateLimited: () => void
    onVPNBlocked: () => void
    onSessionUnavailable: () => void
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
    playSound?: PlaySound
    director?: Director
    signal: AbortSignal
}

export type Globe = {
    readonly tilesCount: number
    setCountry(country: Country): void
    setMapView(view: MapView): void
    takeReward(claimed: ClaimedBonus): void
    setArmed(armed: boolean): void
    setSwitch(name: keyof Switches, on: boolean): void
    setClickHue(hue: number | undefined): void
    capture(): Promise<CapturedFrame>
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
        container: eventTarget,
        country: initialCountry,
        mapView: initialMapView,
        rendering,
        onLeaderboardChange: updateLeaderboard,
        onLoadProgress,
        onRateLimited,
        onVPNBlocked,
        onSessionUnavailable,
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

    let loaded = false

    const graphics = graphicsOf(window.location.search, rendering);
    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget, graphics);
    let mapView: MapView = initialMapView

    let dirty = true
    const invalidate = () => {
        dirty = true
    }

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
        ...blastUniforms(prefersReducedMotion()),
    };

    const pickingUniforms = {pointSize: {value: tilePointSize(camera.zoom, layoutViewport().height) * renderer.getPixelRatio()}}

    const field = new TileField(uniforms, pickingUniforms, geometryData, graphics.tiles);

    const territories = new BorderField(borders, field.size)
    field.setLandmasses(borders.assignment)
    uniforms.landmassData.value = territories.landmassData
    uniforms.landmassCount.value = borders.codes.length

    const picker = new GpuPicker(renderer, field.pickingPoints);
    const ownership = new TileOwnership(field.size);
    const shielded = new TileShields(field.size);

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

    const applyChanges = (changes: OwnerChange[], live = true) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        territories.apply(changes)
        updateLeaderboard(rankCountries(ownership.counts()), live)
        invalidate()
    }

    const showShields = (changes: ShieldChange[]) => {
        if (changes.length === 0) return
        field.setShields(changes)
        invalidate()
    }

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
            reportClaimFailure(e, {onSessionUnavailable})
        })
    }

    let pendingClears: {at: number, tiles: Set<number>, struck: Set<number>}[] = []

    const land = (cleared: number[], struck: number[]) => {
        applyChanges(ownership.applyClears(cleared))
        showShields(shielded.applyClears(cleared))

        const strikes = shielded.applyStrikes(struck)
        showShields(strikes)
        playShields(strikes)
    }

    const flushClears = (upTo: number) => {
        if (pendingClears.length === 0) return false
        const due = pendingClears.filter((clear) => clear.at <= upTo)
        if (due.length === 0) return false
        pendingClears = pendingClears.filter((clear) => clear.at > upTo)
        land(due.flatMap((clear) => [...clear.tiles]), due.flatMap((clear) => [...clear.struck]))
        return true
    }

    let shakeFrom: number | undefined
    const shake = new THREE.Vector3()

    const stopBombs = bomber?.listenForBombs((drop) => {
        const seconds = performance.now() / 1000
        const centre = new THREE.Vector3(drop.point.x, drop.point.y, drop.point.z)
        blasts.start(centre, drop.radius, seconds, drop.tile === undefined)

        if (document.hidden) {
            land(drop.cleared, drop.struck)
        } else if (drop.cleared.length > 0 || drop.struck.length > 0) {
            pendingClears.push({at: seconds + IMPACT_DELAY, tiles: new Set(drop.cleared), struck: new Set(drop.struck)})
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
            reportClaimFailure(e, {onSessionUnavailable})
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

    const canvasPosition = (event: MouseEvent) => {
        const canvas = renderer.domElement
        const rect = canvas.getBoundingClientRect()
        return {
            x: (event.clientX - rect.left) * (canvas.width / rect.width),
            y: (event.clientY - rect.top) * (canvas.height / rect.height),
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

    eventTarget.addEventListener('mousemove', (event: MouseEvent) => {
        pendingPointer = canvasPosition(event)
    }, listenerOptions);

    eventTarget.addEventListener('mouseleave', () => {
        pendingPointer = undefined
        field.setHover(undefined)
        if (armed && !hold.holding) aimAt(undefined)
    }, listenerOptions);

    eventTarget.addEventListener('pointerdown', (event: PointerEvent) => {
        press.begin(event)
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
    }, listenerOptions);

    for (const ending of ['pointerup', 'pointercancel'] as const) {
        eventTarget.addEventListener(ending, (event: PointerEvent) => hold.end(event.pointerId), listenerOptions);
    }

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        if (!loaded || !event.isTrusted) return;

        if (swallowClick) {
            swallowClick = false
            return
        }

        if (press.dragged) return

        const {x, y} = canvasPosition(event)

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
                    reportClaimFailure(e, {onSessionUnavailable})
                })
            return
        }

        if (armed) return

        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return

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
            if (reportClickFailure(e, {onRateLimited, onVPNBlocked, onSessionUnavailable})) playSound("refused")
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
        for (const clear of pendingClears) {
            for (const update of updates) {
                clear.tiles.delete(update.tile)
                clear.struck.delete(update.tile)
            }
        }
        applyChanges(ownership.applyUpdates(updates))

        const moved = shielded.applyUpdates(updates)
        showShields(moved)
        playShields(moved)

        const seconds = performance.now() / 1000
        for (const {tile, clicked, previousCountry, newCountry} of updates) {
            if (!clicked || previousCountry === newCountry || ownClicks.has(tile, country.code, seconds)) continue
            plainClicks.playClick(tile, camera)
        }
    })

    let catchingUp: AbortController | undefined

    const catchUp = async () => {
        catchingUp?.abort()
        const attempt = new AbortController()
        catchingUp = attempt

        ownership.forgetLive()
        shielded.forgetLive()

        const bindings = new Map<number, string>()
        const shields = new Map<number, number>()
        try {
            await pause(CATCH_UP_DELAY_MS, attempt.signal)
            await ownershipsGetter.getCurrentOwnershipsByBatch(
                TILES_PER_BATCH,
                field.size,
                (batch) => {
                    batch.bindings.forEach((owner, tile) => bindings.set(tile, owner))
                    batch.shields.forEach((count, tile) => shields.set(tile, count))
                },
                attempt.signal,
            )
        } catch (e) {
            if (!attempt.signal.aborted) console.error("could not catch up with the map", e)
            return
        }
        if (attempt.signal.aborted) return

        applyChanges(ownership.resync(bindings), false)
        showShields(shielded.resync(shields))
    }

    const stopResumes = updatesListener.listenForResumes(() => void catchUp())

    addDisplayObjects(scene, field.displayPoints, graphics)

    let captureRequests: CaptureRequest[] = []

    const takeCaptureRequests = () => {
        const waiting = captureRequests
        captureRequests = []
        return waiting
    }

    const {stop: stopAnimation} = startAnimation(renderer, scene, camera, uniforms, pickingUniforms, director, () => mapView, () => {
        const was = dirty
        dirty = false
        return was
    }, (seconds) => {
        const boxed = driveBonusBox(seconds)
        const blasting = driveBlasts(seconds)
        outline.update(camera.zoom, renderer.domElement.width, renderer.domElement.height, renderer.getPixelRatio(), mapView)

        if (pendingPointer === undefined) return boxed || blasting
        const {x, y} = pendingPointer
        pendingPointer = undefined
        if (armed) {
            field.setHover(undefined)
            if (!hold.holding) aimAt(surfacePoint(x, y))
            return true
        }
        return field.setHover(picker.pick(camera, x, y)) || boxed || blasting
    }, () => {
        undoShake()
        if (captureRequests.length === 0) return

        const frame = readDrawingBuffer(renderer)
        for (const request of takeCaptureRequests()) request.resolve(frame)
    });

    const globe: Globe = {
        tilesCount: field.size,
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
        capture: () => new Promise<CapturedFrame>((resolve, reject) => {
            if (lifetime.signal.aborted) {
                reject(new Error("the globe is no longer running"))
                return
            }
            captureRequests.push({resolve, reject})
            invalidate()
        }),
        dispose: () => {
            lifetime.abort()

            for (const request of takeCaptureRequests()) {
                request.reject(new Error("the globe was disposed before the frame was read"))
            }

            stopAnimation()
            cleanUpdatesListener()
            stopResumes()
            catchingUp?.abort()
            stopBonuses?.()
            stopBombs?.()

            picker.dispose()
            bonusPointer.dispose()
            blastPointer.dispose()
            blasts.dispose()
            field.dispose()
            territories.dispose()
            outline.dispose()
            bonusBox.dispose()
            enclosures.dispose()
            bonusClicks.dispose()
            plainClicks.dispose()

            cleanup()
        }
    }

    const batches = Math.ceil(field.size / TILES_PER_BATCH)
    let fetched = 0
    const stopFollowingLoad = new AbortController()
    signal.addEventListener("abort", () => lifetime.abort(), {signal: stopFollowingLoad.signal})
    onLoadProgress(0)
    try {
        await ownershipsGetter.getCurrentOwnershipsByBatch(
            TILES_PER_BATCH,
            field.size,
            (ownerships) => {
                applyChanges(ownership.applyBatch(ownerships), false)
                showShields(shielded.applyBatch(ownerships.shields))
                onLoadProgress(Math.min(1, ++fetched / batches))
            },
            lifetime.signal,
        )
        lifetime.signal.throwIfAborted()
    } catch (e) {
        globe.dispose()
        throw e
    } finally {
        stopFollowingLoad.abort()
    }

    loaded = true
    return globe
}

function startAnimation(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.OrthographicCamera,
    uniforms: Uniforms,
    pickingUniforms: {pointSize: THREE.IUniform},
    director: Director | undefined,
    mapView: () => MapView,
    takeChange: () => boolean,
    beforeRender: (seconds: number) => boolean,
    afterRender: () => void,
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

        const turned = director
            ? aim(camera, director(time / 1000))
            : controls.update(spinStep(sinceLastTick)) || moved;

        const {y: height} = renderer.getSize(viewport);
        const ratio = renderer.getPixelRatio();
        uniforms.pointSize.value = displayPointSize(camera.zoom, height, mapView()) * ratio;
        pickingUniforms.pointSize.value = tilePointSize(camera.zoom, height) * ratio;
        uniforms.pixelRatio.value = ratio;

        uniforms.flagPaint.value = flagPaint(camera.zoom, height, mapView());
        uniforms.pixelsPerRadian.value = (renderer.domElement.height / 2) * camera.zoom;

        const moving = beforeRender(time / 1000);
        const changed = takeChange() || moving;

        if (!drawsFrame({turned, changed, at: time, drawnAt, interactingUntil, sinceLastTick})) return;

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
    handlers: {onSessionUnavailable: () => void},
) {
    if (error instanceof BonusLostError) return
    if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else console.error(error)
}

export function reportClickFailure(
    error: unknown,
    handlers: {
        onRateLimited: () => void,
        onVPNBlocked: () => void,
        onSessionUnavailable: () => void,
    },
): boolean {
    if (error instanceof RateLimitedError) handlers.onRateLimited()
    else if (error instanceof VPNBlockedError) handlers.onVPNBlocked()
    else if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else {
        console.error(error)
        return false
    }
    return true
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
