import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, setupScene} from "./scene.ts";
import {loadPointGeometryData} from "./points.ts";
import {GpuPicker} from "./gpuPicking.ts";
import {CapturedFrame, readDrawingBuffer} from "./capture.ts";
import {TileField} from "./tileField.ts";
import {BorderField, loadBorders} from "./borderField.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import {BORDERS_URL} from "./bordersAsset.ts";
import {displayPointSize, flagPaint, tilePointSize} from "./pointSize.ts";
import {regions} from "./atlas.ts";
import {Country} from "../../domain/countries.ts";
import {
    BombDrop,
    Bomber,
    BonusCatch,
    BonusListener,
    BonusLostError,
    BonusOffer,
    OwnershipsGetter,
    RateLimitedError,
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
import {createBonusClickEffects} from "./bonusClickEffects.ts";
import {BonusReward} from "../../domain/bonus.ts";
import {now as monotonicNow} from "../../backends/clickBudget.ts";
import {BlastUniforms, blastUniforms, createBlasts} from "./blasts.ts";
import {IMPACT_DELAY} from "../../domain/blast.ts";
import {HoldToDrop} from "../../domain/holdToDrop.ts";
import {OwnClicks} from "../../domain/ownClicks.ts";
import {PlaySound} from "../sound/soundPlayer.ts";

type Uniforms = BlastUniforms & {
    pointSize: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
    landmassData: THREE.IUniform
    landmassCount: THREE.IUniform
    pixelsPerRadian: THREE.IUniform<number>
    flagPaint: THREE.IUniform
}

/** How long the screen shakes when your own bomb lands, in seconds. */
const SHAKE_SECONDS = 0.5

/** How long a press on the planet has to be held to drop a bomb. */
const HOLD_TO_DROP_SECONDS = 0.7

/** How far a held press may wander before it counts as a drag of the globe. */
const HOLD_TOLERANCE_PX = 6

/** How loud someone else's bomb is, against your own at 1. */
const DISTANT_BOMB_VOLUME = 0.45

/** How long after our own drop a blast in our colours is taken to be it. */
const OWN_DROP_WINDOW_SECONDS = 5

/** How long after our own click a spread or boost on its tile is taken to be it. */
const OWN_CLICK_WINDOW_SECONDS = 3

const TILES_PER_BATCH = 10_000

/**
 * How long before the token lapses the box has to be gone: the claim still has
 * to reach the server, and may have to mint a session on the way.
 */
const CLAIM_MARGIN_MS = 2_000

const textureLoader = new THREE.TextureLoader();

export type GlobeOptions = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    container: HTMLElement
    country: Country
    onLeaderboardChange: (entries: LeaderboardEntry[], live: boolean) => void
    onRateLimited: () => void
    onVPNBlocked: () => void
    onSessionUnavailable: () => void
    /** Somebody on the planet caught a box — this client included. */
    onBonusTaken: (taken: BonusCatch) => void
    /** What this client won, once the server has agreed to it. */
    onBonusWon: (reward: BonusReward) => void
    /** This client closed a shape with its enclose bonus, which has this many left. */
    onShapeClosed: (shapesLeft: number) => void
    /** Absent for a backend with no bonus feed, which draws no boxes at all. */
    bonusListener?: BonusListener
    /** Absent for a backend with no bombs: a bomb won is then never armed. */
    bomber?: Bomber
    /** A bomb landed somewhere on the planet — this client's included. */
    onBombDropped: (drop: BombDrop) => void
    /** The bomb this client held is gone: dropped, or held too long. */
    onBombSpent: () => void
    /** Read for the globe's whole life, so it must not change identity. */
    playSound?: PlaySound
    signal: AbortSignal
}

export type Globe = {
    readonly tilesCount: number
    setCountry(country: Country): void
    /** Plays out a reward as if a box had just been caught and the server had
     *  answered with it. The click path's own step, exposed for dev tooling. */
    takeReward(reward: BonusReward): void
    /** The globe as it is framed right now, resolved on the next frame — the
     *  only tick the drawing buffer can be read from. See capture.ts. */
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
        onLeaderboardChange: updateLeaderboard,
        onRateLimited,
        onVPNBlocked,
        onSessionUnavailable,
        onBonusTaken,
        onBonusWon,
        onShapeClosed,
        bonusListener,
        bomber,
        onBombDropped,
        onBombSpent,
        playSound = () => {},
        signal,
    } = options

    // Both blobs before a single GPU resource exists, so an abandoned load never
    // opens a context, and in parallel because neither needs the other.
    const [geometryData, borders] = await Promise.all([
        loadPointGeometryData(signal),
        loadBorders(BORDERS_URL, signal),
    ]);
    if (signal.aborted) throw new DOMException("globe load aborted", "AbortError");

    const lifetime = new AbortController();
    const listenerOptions = {signal: lifetime.signal};

    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget);
    const uniforms: Uniforms = {
        pointSize: {value: displayPointSize(camera.zoom, layoutViewport().height)},
        atlasTexture: {value: textureLoader.load(ATLAS_URL)},
        atlasTextureSize: {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)},
        landmassData: {value: null},
        landmassCount: {value: 1},
        pixelsPerRadian: {value: 1},
        flagPaint: {value: flagPaint(camera.zoom, layoutViewport().height)},
        ...blastUniforms(prefersReducedMotion()),
    };

    // The picking pass keeps the true tile size: the display discs are widened
    // to cover the ground while the coarse flag is painted through them, and
    // overlapping discs would hand a click to whichever neighbour drew last.
    const pickingUniforms = {pointSize: {value: tilePointSize(camera.zoom, layoutViewport().height)}}

    const field = new TileField(uniforms, pickingUniforms, geometryData);

    const territories = new BorderField(borders, field.size)
    field.setLandmasses(borders.assignment)
    uniforms.landmassData.value = territories.landmassData
    uniforms.landmassCount.value = borders.codes.length

    const picker = new GpuPicker(renderer, field.pickingPoints);
    const ownership = new TileOwnership(field.size);

    let country: Country = initialCountry;

    const bonusBox = createBonusBox()
    scene.add(bonusBox.object)

    // Zoomed in the box is nearly always outside the frame, so without this a
    // zoomed player never learns one was theirs.
    const bonusPointer = createBonusPointer(eventTarget)

    // Every shape anyone closes, drawn on every screen: the tiles alone flip
    // with no reason given.
    const enclosures = createEnclosureEffects(geometryData.positions)
    scene.add(enclosures.object)

    // And every click made under a spread or a triple clicks bonus, anyone's.
    const bonusClicks = createBonusClickEffects(geometryData.positions)
    scene.add(bonusClicks.object)

    // The box on screen and the token that redeems it, held together: a box
    // caught is only worth something with the token it arrived with.
    let offered: BonusOffer | undefined

    // A spread or a boost is broadcast to everyone with no word of whose it is,
    // and only the player who made it hears it: its tile is one they just clicked.
    const ownClicks = new OwnClicks(OWN_CLICK_WINDOW_SECONDS)

    // The server decides when a box appears and who sees it, so nothing here
    // schedules one: the stream says so, and the seed it sends is what draws
    // the orbit. The box ends itself, so there is no matching "hide".
    const stopBonuses = bonusListener?.listenForBonuses({
        onOffered: (offer) => {
            offered = offer
            // The animation loop's clock is `performance.now()` in seconds, the
            // same clock the offer's deadline is on.
            bonusBox.spawn(offer.seed, (offer.expiresAt - CLAIM_MARGIN_MS) / 1000)
            playSound("bonusSpawn")
        },
        onTaken: (taken) => onBonusTaken(taken),
        onEnclosed: (enclosure) => {
            enclosures.play(enclosure)
            if (enclosure.yours) {
                playSound("enclose")
                onShapeClosed(enclosure.yours.shapesLeft)
            }
        },
        onSpread: (spread) => {
            bonusClicks.playSpread(spread)
            if (ownClicks.has(spread.tile, spread.countryId, performance.now() / 1000)) playSound("spread")
        },
    })

    const driveBonusBox = (seconds: number) => {
        enclosures.update(seconds, camera, renderer.domElement.height)
        bonusClicks.update(seconds, camera, renderer.domElement.height)
        bonusBox.update(seconds, camera)
        bonusPointer.update(bonusBox.flying ? bonusBox.object.position : undefined, camera)
    }

    // `live` tells the board apart from its own footing: everything that lands
    // while the player watches is news, the map it was handed at the start is not.
    const applyChanges = (changes: OwnerChange[], live = true) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        territories.apply(changes)
        updateLeaderboard(rankCountries(ownership.counts()), live)
    }

    const blasts = createBlasts(uniforms, uniforms.pixelsPerRadian)
    scene.add(blasts.object)
    // A blast is usually on the side of the planet nobody is looking at.
    const blastPointer = createBonusPointer(eventTarget, "blast")

    // The bomb this client holds, from the answer to its claim until it is
    // dropped or lapses.
    //
    // Aiming follows the sphere under the cursor, not the tile picker: the
    // picker finds nothing between tiles or over the sea, and a ring that
    // followed it blinked off and jumped from tile to tile as the mouse moved.
    let armed: {endsAt: number, radius: number} | undefined

    // See domain/holdToDrop.ts: a bomb goes on a press held still, never a click.
    const hold = new HoldToDrop<THREE.Vector3>({holdSeconds: HOLD_TO_DROP_SECONDS, tolerancePx: HOLD_TOLERANCE_PX})

    // The click that follows a press ends nothing while a bomb is involved.
    let swallowClick = false

    const cancelCharge = () => {
        hold.cancel()
        blasts.setCharge(0)
    }

    const disarm = () => {
        armed = undefined
        cancelCharge()
        blasts.setAim(undefined, 0)
        eventTarget.classList.remove("viewer-canvas--armed")
        onBombSpent()
    }

    const aimAt = (point: THREE.Vector3 | undefined) => {
        if (!armed) return
        blasts.setAim(point, armed.radius)
    }

    // What the server granted for a caught box. A bomb is armed here; every
    // reward is then announced the same way.
    const takeReward = (reward: BonusReward) => {
        if (reward.kind === "bomb" && bomber) {
            armed = {endsAt: monotonicNow() + reward.seconds * 1000, radius: reward.radius}
            eventTarget.classList.add("viewer-canvas--armed")
        }
        onBonusWon(reward)
    }

    // When this client last dropped one, until its broadcast comes back: the
    // server picks the tile, so the next blast in our colours is ours, for the shake.
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

    // Cleared tiles wait for the blast to hit them, so the ground goes when the
    // bomb explodes rather than when the message arrives.
    let pendingClears: {at: number, tiles: Set<number>}[] = []

    const flushClears = (upTo: number) => {
        if (pendingClears.length === 0) return
        const due = pendingClears.filter((clear) => clear.at <= upTo)
        if (due.length === 0) return
        pendingClears = pendingClears.filter((clear) => clear.at > upTo)
        applyChanges(ownership.applyClears(due.flatMap((clear) => [...clear.tiles])))
    }

    let shakeFrom: number | undefined
    const shake = new THREE.Vector3()

    const stopBombs = bomber?.listenForBombs((drop) => {
        // The animation loop's own clock, so the blast starts on this frame.
        const seconds = performance.now() / 1000
        const centre = new THREE.Vector3(drop.point.x, drop.point.y, drop.point.z)
        blasts.start(centre, drop.radius, seconds, drop.tile === undefined)

        // A hidden tab draws no frames, so nothing would ever reach the impact.
        if (document.hidden) {
            applyChanges(ownership.applyClears(drop.cleared))
        } else if (drop.cleared.length > 0) {
            pendingClears.push({at: seconds + IMPACT_DELAY, tiles: new Set(drop.cleared)})
        }

        const own = ownDropAt !== undefined && drop.countryId === country.code && seconds - ownDropAt < OWN_DROP_WINDOW_SECONDS
        if (own) {
            ownDropAt = undefined
            shakeFrom = seconds + IMPACT_DELAY
        }
        // The synth waits `IMPACT_DELAY` itself, so the boom lands with the tiles.
        // A drop with no tile under it landed in the ocean, and splashes.
        playSound("bomb", {volume: own ? 1 : DISTANT_BOMB_VOLUME, onWater: drop.tile === undefined})
        onBombDropped(drop)
    })

    const driveBlasts = (seconds: number) => {
        blasts.update(seconds, camera)
        blastPointer.update(blasts.newest(seconds), camera)
        flushClears(seconds)

        if (armed && monotonicNow() >= armed.endsAt) disarm()

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
                // Undone after the frame is drawn, or OrbitControls would read
                // it back as the player turning the globe.
                const amplitude = 0.015 * (1 - s / SHAKE_SECONDS) ** 2 / camera.zoom
                shake.set(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5).multiplyScalar(amplitude)
                camera.position.add(shake)
            }
        }
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

    /** Where on the globe the canvas point (x, y) is, or undefined off the planet. */
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
        if (!armed || !event.isTrusted || !event.isPrimary || event.button !== 0) return

        const {x, y} = canvasPosition(event)
        const point = surfacePoint(x, y)
        if (!point) return

        hold.begin(event.pointerId, event.clientX, event.clientY, performance.now() / 1000, point)
        swallowClick = true
        aimAt(point)
    }, listenerOptions);

    eventTarget.addEventListener('pointermove', (event: PointerEvent) => {
        hold.move(event.pointerId, event.clientX, event.clientY)
    }, listenerOptions);

    for (const ending of ['pointerup', 'pointercancel'] as const) {
        eventTarget.addEventListener(ending, (event: PointerEvent) => hold.end(event.pointerId), listenerOptions);
    }

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        if (!event.isTrusted) return;

        if (swallowClick) {
            swallowClick = false
            return
        }

        const {x, y} = canvasPosition(event)

        // The box sits above the tile shell, so it has to be asked first:
        // otherwise a click meant for it paints whatever tile is behind it.
        // The box pops on the click rather than on the answer: the server
        // addressed this box to this client, so the only way to lose it now is
        // to have let it lapse, and making the player watch a round trip before
        // anything happens would cost every catch its snap.
        // A box whose token is about to lapse is let go rather than popped: the
        // frame that would have ended it may not have been drawn yet.
        if (offered && monotonicNow() >= offered.expiresAt - CLAIM_MARGIN_MS) {
            offered = undefined
            bonusBox.hide()
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

        // Holding a bomb, a click claims nothing: the bomb goes on a held press.
        if (armed) return

        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return

        if (!regions.get(country.code)) {
            warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
            return
        }

        const {changes, claim} = ownership.applyOptimistic(tile, country.code)
        applyChanges(changes)
        playSound("click")
        ownClicks.record(tile, country.code, performance.now() / 1000)

        tileClicker.clickTile(tile, country.code).catch((e) => {
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

        renderer.setSize(width, height);
    };
    window.addEventListener('resize', resizeListener, listenerOptions);

    ownershipsGetter.getCurrentOwnershipsByBatch(
        TILES_PER_BATCH,
        field.size,
        (ownerships) => applyChanges(ownership.applyBatch(ownerships), false),
        lifetime.signal,
    ).catch((e) => {
        if (lifetime.signal.aborted) return
        console.error("Failed to fetch initial ownerships", e)
    })

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch((updates: Update[]) => {
        // An update reaches this client after the blast it follows, so it wins
        // its tile: that tile is taken out of the waiting clear, and the rest of
        // the crater still goes on impact.
        for (const clear of pendingClears) {
            for (const update of updates) clear.tiles.delete(update.tile)
        }
        applyChanges(ownership.applyUpdates(updates))

        const seconds = performance.now() / 1000
        for (const update of updates) {
            if (!update.boosted) continue
            bonusClicks.playBoost(update.tile)
            if (ownClicks.has(update.tile, update.newCountry, seconds)) playSound("boost")
        }
    })

    addDisplayObjects(scene, field.displayPoints)

    let captureRequests: CaptureRequest[] = []

    const takeCaptureRequests = () => {
        const waiting = captureRequests
        captureRequests = []
        return waiting
    }

    const {stop: stopAnimation} = startAnimation(renderer, scene, camera, uniforms, pickingUniforms, (seconds) => {
        driveBonusBox(seconds)
        driveBlasts(seconds)

        if (pendingPointer === undefined) return
        const {x, y} = pendingPointer
        pendingPointer = undefined
        // Holding a bomb, the ring is the only hover: no tile pick, no tile
        // highlight blinking on and off under it.
        if (armed) {
            field.setHover(undefined)
            if (!hold.holding) aimAt(surfacePoint(x, y))
            return
        }
        field.setHover(picker.pick(camera, x, y))
    }, () => {
        undoShake()
        if (captureRequests.length === 0) return

        // Still inside the frame that drew it, which is the whole reason this
        // hook exists rather than a method anyone could call.
        const frame = readDrawingBuffer(renderer)
        for (const request of takeCaptureRequests()) request.resolve(frame)
    });

    return {
        tilesCount: field.size,
        setCountry: (newCountry: Country) => {
            country = newCountry
        },
        takeReward,
        capture: () => new Promise<CapturedFrame>((resolve, reject) => {
            if (lifetime.signal.aborted) {
                reject(new Error("the globe is no longer running"))
                return
            }
            captureRequests.push({resolve, reject})
        }),
        dispose: () => {
            lifetime.abort()

            for (const request of takeCaptureRequests()) {
                request.reject(new Error("the globe was disposed before the frame was read"))
            }

            stopAnimation()
            cleanUpdatesListener()
            stopBonuses?.()
            stopBombs?.()

            picker.dispose()
            bonusPointer.dispose()
            blastPointer.dispose()
            blasts.dispose()
            field.dispose()
            territories.dispose()
            bonusBox.dispose()
            enclosures.dispose()
            bonusClicks.dispose()

            cleanup()
        }
    }
}

function startAnimation(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.OrthographicCamera,
    uniforms: Uniforms,
    pickingUniforms: {pointSize: THREE.IUniform},
    beforeRender: (seconds: number) => void,
    afterRender: () => void,
): {stop: () => void} {
    const starfield = createStarfield();

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.minZoom = MIN_ZOOM;
    controls.maxZoom = MAX_ZOOM;
    controls.panSpeed = 0.1;
    controls.enableDamping = true;

    controls.addEventListener('change', () => {
        controls.autoRotate = camera.zoom <= RESTING_ZOOM;

        controls.rotateSpeed = (1 / camera.zoom) / 1.5;
    });

    renderer.setAnimationLoop((time: number) => {
        controls.update();
        beforeRender(time / 1000);
        starfield.render(renderer, camera, () => renderer.render(scene, camera));
        // After the starfield's pass, not inside it: the sky is drawn first and
        // the globe over it, so the buffer only holds the whole frame here.
        afterRender();
        uniforms.pointSize.value = displayPointSize(camera.zoom, renderer.domElement.height);
        pickingUniforms.pointSize.value = tilePointSize(camera.zoom, renderer.domElement.height);

        // The coarse layer owns the frame until a tile is big enough to be aimed
        // at. A landmass is painted as its holder's flag until its own tiles are
        // big enough to be flags in their own right.
        uniforms.flagPaint.value = flagPaint(camera.zoom, renderer.domElement.height);
        // The globe's radius is 1, so an arc of one radian is half the viewport
        // at zoom 1.
        uniforms.pixelsPerRadian.value = (renderer.domElement.height / 2) * camera.zoom;
    });

    return {
        stop: () => {
            renderer.setAnimationLoop(null);
            controls.dispose();
            // Its scene is not the one `disposeScene` walks, so it goes here.
            starfield.dispose();
        },
    };
}

function prefersReducedMotion(): boolean {
    return typeof window.matchMedia === "function"
        && window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

/**
 * A box that got away is not worth a dialog — it lapsed, or the server had
 * already given it to nobody. A session that could not be minted still is, since
 * that is the player's clicks stopping too.
 */
export function reportClaimFailure(
    error: unknown,
    handlers: {onSessionUnavailable: () => void},
) {
    if (error instanceof BonusLostError) return
    if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else console.error(error)
}

/** Whether the server refused the click, as opposed to it never getting an answer. */
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
