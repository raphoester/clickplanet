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

type Uniforms = {
    pointSize: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
    landmassData: THREE.IUniform
    landmassCount: THREE.IUniform
    pixelsPerRadian: THREE.IUniform
    flagPaint: THREE.IUniform
}

const TILES_PER_BATCH = 10_000

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
    signal: AbortSignal
}

export type Globe = {
    readonly tilesCount: number
    setCountry(country: Country): void
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

    // `live` tells the board apart from its own footing: everything that lands
    // while the player watches is news, the map it was handed at the start is not.
    const applyChanges = (changes: OwnerChange[], live = true) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        territories.apply(changes)
        updateLeaderboard(rankCountries(ownership.counts()), live)
    }

    const canvasPosition = (event: MouseEvent) => {
        const canvas = renderer.domElement
        const rect = canvas.getBoundingClientRect()
        return {
            x: (event.clientX - rect.left) * (canvas.width / rect.width),
            y: (event.clientY - rect.top) * (canvas.height / rect.height),
        }
    }

    let pendingPointer: {x: number, y: number} | undefined

    eventTarget.addEventListener('mousemove', (event: MouseEvent) => {
        pendingPointer = canvasPosition(event)
    }, listenerOptions);

    eventTarget.addEventListener('mouseleave', () => {
        pendingPointer = undefined
        field.setHover(undefined)
    }, listenerOptions);

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        if (!event.isTrusted) return;

        const {x, y} = canvasPosition(event)
        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return

        if (!regions.get(country.code)) {
            warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
            return
        }

        const {changes, claim} = ownership.applyOptimistic(tile, country.code)
        applyChanges(changes)

        tileClicker.clickTile(tile, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            applyChanges(ownership.rollback(claim))
            reportClickFailure(e, {onRateLimited, onVPNBlocked, onSessionUnavailable})
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

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch(
        (updates: Update[]) => applyChanges(ownership.applyUpdates(updates)))

    addDisplayObjects(scene, field.displayPoints)

    let captureRequests: CaptureRequest[] = []

    const takeCaptureRequests = () => {
        const waiting = captureRequests
        captureRequests = []
        return waiting
    }

    const {stop: stopAnimation} = startAnimation(renderer, scene, camera, uniforms, pickingUniforms, () => {
        if (pendingPointer === undefined) return
        const {x, y} = pendingPointer
        pendingPointer = undefined
        field.setHover(picker.pick(camera, x, y))
    }, () => {
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

            picker.dispose()
            field.dispose()
            territories.dispose()

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
    beforeRender: () => void,
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

    renderer.setAnimationLoop(() => {
        controls.update();
        beforeRender();
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

export function reportClickFailure(
    error: unknown,
    handlers: {
        onRateLimited: () => void,
        onVPNBlocked: () => void,
        onSessionUnavailable: () => void,
    },
) {
    if (error instanceof RateLimitedError) handlers.onRateLimited()
    else if (error instanceof VPNBlockedError) handlers.onVPNBlocked()
    else if (error instanceof SessionUnavailableError) handlers.onSessionUnavailable()
    else console.error(error)
}
