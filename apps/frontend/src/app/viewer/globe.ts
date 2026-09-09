import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, setupScene} from "./scene.ts";
import {loadPointGeometryData} from "./points.ts";
import {GpuPicker} from "./gpuPicking.ts";
import {TileField} from "./tileField.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import {tilePointSize} from "./pointSize.ts";
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
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts";
import {OwnerChange, TileOwnership} from "../../domain/tileOwnership.ts";
import {warnOnce} from "../../domain/warnOnce.ts";

type Uniforms = {
    pointSize: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
}

const TILES_PER_BATCH = 10_000

const textureLoader = new THREE.TextureLoader();

export type GlobeOptions = {
    tileClicker: TileClicker
    ownershipsGetter: OwnershipsGetter
    updatesListener: UpdatesListener
    container: HTMLElement
    country: Country
    onLeaderboardChange: (entries: LeaderboardEntry[]) => void
    onRateLimited: () => void
    onVPNBlocked: () => void
    signal: AbortSignal
}

export type Globe = {
    readonly tilesCount: number
    setCountry(country: Country): void
    dispose(): void
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
        signal,
    } = options

    const geometryData = await loadPointGeometryData(signal);
    if (signal.aborted) throw new DOMException("globe load aborted", "AbortError");

    const lifetime = new AbortController();
    const listenerOptions = {signal: lifetime.signal};

    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget);
    const uniforms: Uniforms = {
        pointSize: {value: tilePointSize(camera.zoom, window.innerHeight)},
        atlasTexture: {value: textureLoader.load(ATLAS_URL)},
        atlasTextureSize: {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)},
    };

    const field = new TileField(uniforms, geometryData);
    const picker = new GpuPicker(renderer, field.pickingPoints);
    const ownership = new TileOwnership(field.size);

    let country: Country = initialCountry;

    const applyChanges = (changes: OwnerChange[]) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        updateLeaderboard(rankCountries(ownership.counts()))
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

        tileClicker.clickTile(tile, country.code).catch((e) => {
            if (lifetime.signal.aborted) return
            reportClickFailure(e, {onRateLimited, onVPNBlocked})
        })

        applyChanges(ownership.applyUpdates([{
            tile,
            previousCountry: ownership.ownerOf(tile),
            newCountry: country.code,
        }]))
    }, listenerOptions);

    const resizeListener = () => {
        const width = window.innerWidth;
        const height = window.innerHeight;

        camera.left = -cameraSize * (width / height);
        camera.right = cameraSize * (width / height);
        camera.updateProjectionMatrix();

        renderer.setSize(width, height);
    };
    window.addEventListener('resize', resizeListener, listenerOptions);

    ownershipsGetter.getCurrentOwnershipsByBatch(
        TILES_PER_BATCH,
        field.size,
        (ownerships) => applyChanges(ownership.applyBatch(ownerships)),
        lifetime.signal,
    ).catch((e) => {
        if (lifetime.signal.aborted) return
        console.error("Failed to fetch initial ownerships", e)
    })

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch(
        (updates: Update[]) => applyChanges(ownership.applyUpdates(updates)))

    addDisplayObjects(scene, field.displayPoints)

    const stopAnimation = startAnimation(renderer, scene, camera, uniforms, () => {
        if (pendingPointer === undefined) return
        const {x, y} = pendingPointer
        pendingPointer = undefined
        field.setHover(picker.pick(camera, x, y))
    });

    return {
        tilesCount: field.size,
        setCountry: (newCountry: Country) => {
            country = newCountry
        },
        dispose: () => {
            lifetime.abort()

            stopAnimation()
            cleanUpdatesListener()

            picker.dispose()
            field.dispose()

            cleanup()
        }
    }
}

function startAnimation(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.OrthographicCamera,
    uniforms: Uniforms,
    beforeRender: () => void,
): () => void {
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.minZoom = 1;
    controls.maxZoom = 50;
    controls.panSpeed = 0.1;
    controls.enableDamping = true;

    controls.addEventListener('change', () => {
        controls.autoRotate = camera.zoom === 1;

        controls.rotateSpeed = (1 / camera.zoom) / 1.5;
    });

    renderer.setAnimationLoop(() => {
        controls.update();
        beforeRender();
        renderer.render(scene, camera);
        uniforms.pointSize.value = tilePointSize(camera.zoom, renderer.domElement.height);
    });

    return () => {
        renderer.setAnimationLoop(null);
        controls.dispose();
    };
}

export function reportClickFailure(
    error: unknown,
    handlers: {onRateLimited: () => void, onVPNBlocked: () => void},
) {
    if (error instanceof RateLimitedError) handlers.onRateLimited()
    else if (error instanceof VPNBlockedError) handlers.onVPNBlocked()
    else console.error(error)
}
