import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, setupScene} from "./scene.ts";
import {loadPointGeometryData} from "./points.ts";
import {GpuPicker} from "./gpuPicking.ts";
import {TileField} from "./tileField.ts";
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts";
import {regions} from "./atlas.ts";
import {Country} from "../../domain/countries.ts";
import {OwnershipsGetter, TileClicker, Update, UpdatesListener} from "../../backends/backend.ts";
import {LeaderboardEntry, rankCountries} from "../../domain/leaderboard.ts";
import {OwnerChange, TileOwnership} from "../../domain/tileOwnership.ts";
import {warnOnce} from "../../domain/warnOnce.ts";

type Uniforms = {
    zoom: THREE.IUniform,
    resolution: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
}

const TILES_PER_BATCH = 10_000

const textureLoader = new THREE.TextureLoader();

export type EffectHandle = Awaited<ReturnType<typeof effect>>

export async function effect(
    tileClicker: TileClicker,
    ownershipsGetter: OwnershipsGetter,
    updatesListener: UpdatesListener,
    updateLeaderboard: (data: LeaderboardEntry[]) => void,
    eventTarget: HTMLElement,
    countryState: Country,
    signal: AbortSignal,
) {
    // Fetched before anything is allocated, so a run abandoned during the
    // download (StrictMode's throwaway first mount, or an unmount) never opens a
    // WebGL context in the first place.
    const geometryData = await loadPointGeometryData(signal);
    if (signal.aborted) throw new DOMException("effect aborted", "AbortError");

    /** Aborted by cleanup(); detaches every listener this run registered. */
    const lifetime = new AbortController();
    const listenerOptions = {signal: lifetime.signal};

    const {scene, camera, cameraSize, renderer, cleanup} = setupScene(eventTarget);
    const uniforms: Uniforms = {
        zoom: {value: 1.0},
        resolution: {value: new THREE.Vector2(window.innerWidth, window.innerHeight)},
        atlasTexture: {value: textureLoader.load(ATLAS_URL)},
        atlasTextureSize: {value: new THREE.Vector2(ATLAS_SIZE.width, ATLAS_SIZE.height)},
    };

    const field = new TileField(uniforms, geometryData);
    const picker = new GpuPicker(renderer, field.pickingPoints);
    const ownership = new TileOwnership(field.size);
    console.log("running with", field.size, "points");

    let country: Country = countryState;

    const applyChanges = (changes: OwnerChange[]) => {
        if (changes.length === 0) return
        field.setOwners(changes)
        updateLeaderboard(rankCountries(ownership.counts()))
    }

    /** Event coordinates in the canvas's own pixels, whatever the pixel ratio. */
    const canvasPosition = (event: MouseEvent) => {
        const canvas = renderer.domElement
        const rect = canvas.getBoundingClientRect()
        return {
            x: (event.clientX - rect.left) * (canvas.width / rect.width),
            y: (event.clientY - rect.top) * (canvas.height / rect.height),
        }
    }

    /**
     * Where the cursor was last seen, resolved to a tile once per frame rather
     * than once per event. A pick costs a render and a synchronous GPU read, and
     * mousemove fires far more often than the screen refreshes.
     */
    let pendingPointer: {x: number, y: number} | undefined

    eventTarget.addEventListener('mousemove', (event: MouseEvent) => {
        pendingPointer = canvasPosition(event)
    }, listenerOptions);

    eventTarget.addEventListener('mouseleave', () => {
        pendingPointer = undefined
        field.setHover(undefined)
    }, listenerOptions);

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        /** protection from element.dispatchEvent(e) */
        if (!event.isTrusted) return;

        const {x, y} = canvasPosition(event)
        const tile = picker.pick(camera, x, y)
        if (tile === undefined) return

        if (!regions.get(country.code)) {
            warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
            return
        }

        tileClicker.clickTile(tile, country.code).catch(console.error)

        /** Painted straight away; the server's echo confirms it later. */
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
        uniforms.resolution.value.set(width, height);
    };
    // resize is a window event and cannot be captured by the eventTarget
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
        updateCountry: (newCountry: Country) => {
            country = newCountry
        },
        tilesCount: field.size,
        /** Safe to call at any point after effect() resolves, and only once. */
        cleanup: () => {
            // Detaches every listener this run registered. Replaces the old
            // clone-and-swap trick, which also destroyed the canvas a concurrent
            // StrictMode run had appended to the same container.
            lifetime.abort()

            stopAnimation()
            cleanUpdatesListener()

            // The picking points live only in the picker's own scene, so
            // setupScene's cleanup never sees them.
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
    // Gives a smooth effect to the action of rotating the sphere with the mouse
    controls.enableDamping = true;

    controls.addEventListener('change', () => {
        controls.autoRotate = camera.zoom === 1;

        // Decreases the speed at which the user can rotate the sphere with the mouse the more he zooms in
        controls.rotateSpeed = (1 / camera.zoom) / 1.5;
    });

    // setAnimationLoop rather than a self-scheduling requestAnimationFrame: the
    // latter cannot be stopped, so every remount left another loop rendering
    // through a disposed renderer.
    renderer.setAnimationLoop(() => {
        controls.update();
        beforeRender();
        renderer.render(scene, camera);
        uniforms.zoom.value = camera.zoom;
    });

    return () => {
        renderer.setAnimationLoop(null);
        controls.dispose();
    };
}
