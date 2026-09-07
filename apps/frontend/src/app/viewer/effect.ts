import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, disposeMaterial, setupScene} from "./scene.ts";
import {createPoints, loadPointGeometryData} from "./points.ts";
import {actOnPick} from "./gpuPicking.ts";
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
        atlasTexture: {value: textureLoader.load(`/static/countries/atlas.png?ts=${Date.now()}`)},
        atlasTextureSize: {value: new THREE.Vector2(1300, 1232)}, // TODO: retrieve the size from the texture itself
    };

    const {pickingPoints, displayPoints, size} = createPoints(uniforms, geometryData);
    console.log("running with", size, "points");

    // 0 means no region is selected, by default it's like that for the whole map
    const generateDefaultRegionVector = (size: number) => {
        return new Float32Array(size * 4).fill(0);
    }

    displayPoints.geometry.setAttribute('regionVector', new THREE.BufferAttribute(generateDefaultRegionVector(size), 4));

    const ownership = new TileOwnership(size)

    /** Paints the changed tiles and re-ranks the leaderboard from the store. */
    const applyChanges = (changes: OwnerChange[]) => {
        if (changes.length === 0) return

        const attribute = displayPoints.geometry.getAttribute('regionVector')
        const regionVectors = attribute.array as Float32Array
        for (const {tile, country} of changes) {
            const region = regions.get(country)
            if (!region) {
                warnOnce(`No sprite region for country "${country}", leaving its tiles blank`)
                continue
            }

            const offset = (tile - 1) * 4
            regionVectors[offset] = region.x
            regionVectors[offset + 1] = region.y
            regionVectors[offset + 2] = region.width
            regionVectors[offset + 3] = region.height
        }
        attribute.needsUpdate = true

        updateLeaderboard(rankCountries(ownership.counts()))
    }

    let country: Country = countryState;

    function actOnPick_(
        event: MouseEvent,
        callback: (id: number) => void,
        nullCallback?: () => void, // if no point is selected
    ) {
        return actOnPick(renderer, camera, event, pickingPoints, callback, nullCallback);
    }

    eventTarget.addEventListener('mousemove', (event: MouseEvent) => {
        actOnPick_(event,
            id => updateHoverEffect(displayPoints.geometry, id),
            () => updateHoverEffect(displayPoints.geometry)
        )
    }, listenerOptions);

    eventTarget.addEventListener('click', (event: MouseEvent) => {
        /**
         * protection from element.dispatchEvent(e)
         */
        if (!event.isTrusted) return;
        actOnPick_(event, id => {
            const region = regions.get(country.code)
            if (!region) {
                warnOnce(`No sprite region for country "${country.code}", ignoring the click`)
                return
            }

            tileClicker.clickTile(id, country.code).catch(console.error)

            /** Painted straight away; the server's echo confirms it later. */
            applyChanges(ownership.applyUpdates([{
                tile: id,
                previousCountry: ownership.ownerOf(id),
                newCountry: country.code,
            }]))
        });
    }, listenerOptions);

    const resizeListener = () => {
        const width = window.innerWidth;
        const height = window.innerHeight;

        const aspect = width / height;

        camera.left = -cameraSize * aspect;
        camera.right = cameraSize * aspect;
        // not needed but kept in mind in case the camera is resized
        // camera.top = cameraSize;
        // camera.bottom = -cameraSize;
        camera.updateProjectionMatrix();

        renderer.setSize(width, height);

        uniforms.resolution.value.set(width, height);
    };
    // resize is a window event and cannot be captured by the eventTarget
    window.addEventListener('resize', resizeListener, listenerOptions);

    const tilesPerBatch = 10_000
    ownershipsGetter.getCurrentOwnershipsByBatch(
        tilesPerBatch,
        size,
        (ownerships) => applyChanges(ownership.applyBatch(ownerships)),
        lifetime.signal,
    ).catch((e) => {
        if (lifetime.signal.aborted) return
        console.error("Failed to fetch initial ownerships", e)
    })

    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch(
        (updates: Update[]) => applyChanges(ownership.applyUpdates(updates)))

    addDisplayObjects(scene, displayPoints)
    const stopAnimation = startAnimation(renderer, scene, camera, uniforms);

    return {
        updateCountry: (newCountry: Country) => {
            country = newCountry
        },
        country: country,
        tilesCount: size,
        /** Safe to call at any point after effect() resolves, and only once. */
        cleanup: () => {
            // Detaches every listener this run registered. Replaces the old
            // clone-and-swap trick, which also destroyed the canvas a concurrent
            // StrictMode run had appended to the same container.
            lifetime.abort()

            stopAnimation()
            cleanUpdatesListener()

            // pickingPoints only ever lives in the throwaway scene gpuPicking
            // builds per event, so setupScene's cleanup never sees it.
            pickingPoints.geometry.dispose()
            disposeMaterial(pickingPoints.material as THREE.Material)

            cleanup()
        }
    }
}

function startAnimation(
    renderer: THREE.WebGLRenderer,
    scene: THREE.Scene,
    camera: THREE.OrthographicCamera,
    uniforms: Uniforms,
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
        renderer.render(scene, camera);
        updateUniforms(camera, uniforms);
    });

    return () => {
        renderer.setAnimationLoop(null);
        controls.dispose();
    };
}

function updateUniforms(camera: THREE.OrthographicCamera, uniforms: Uniforms) {
    uniforms.zoom.value = camera.zoom;
}

export function updateHoverEffect(geometry: THREE.BufferGeometry, hoveredId?: number) {
    const size = geometry.attributes.hover.array.length;
    const newHovered = new Float32Array(size).fill(0);
    if (hoveredId) {
        const arrayIdIndexedOnZero = hoveredId - 1
        newHovered[arrayIdIndexedOnZero] = 1;
    }

    geometry.setAttribute('hover', new THREE.BufferAttribute(newHovered, 1));
    geometry.attributes.hover.needsUpdate = true;
}
