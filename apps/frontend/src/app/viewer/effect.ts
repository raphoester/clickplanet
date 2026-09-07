import * as THREE from "three";
import {OrbitControls} from "three/addons/controls/OrbitControls.js";
import {addDisplayObjects, disposeMaterial, setupScene} from "./scene.ts";
import {createPoints, loadPointGeometryData} from "./points.ts";
import {actOnPick} from "./gpuPicking.ts";
import {regions} from "./atlas.ts";
import {Countries, Country} from "../countries.ts";
import {OwnershipsGetter, TileClicker, Update, UpdatesListener} from "../../backends/backend.ts";
import {Leaderboard} from "./leaderboard.ts";

type Uniforms = {
    zoom: THREE.IUniform,
    resolution: THREE.IUniform
    atlasTexture: THREE.IUniform
    atlasTextureSize: THREE.IUniform
}

const textureLoader = new THREE.TextureLoader();

const warnedAbout = new Set<string>()

function warnOnce(message: string) {
    if (warnedAbout.has(message)) return
    warnedAbout.add(message)
    console.warn(message)
}

export type EffectHandle = Awaited<ReturnType<typeof effect>>

export async function effect(
    tileClicker: TileClicker,
    ownershipsGetter: OwnershipsGetter,
    updatesListener: UpdatesListener,
    updateLeaderboard: (data: { country: Country, tiles: number }[]) => void,
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

    const leaderboard = new Leaderboard(updateLeaderboard)

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

            const arrayIdIndexedOnZero = id - 1
            const arr = displayPoints.geometry.getAttribute('regionVector').array as Float32Array
            arr[arrayIdIndexedOnZero * 4] = region.x
            arr[arrayIdIndexedOnZero * 4 + 1] = region.y
            arr[arrayIdIndexedOnZero * 4 + 2] = region.width
            arr[arrayIdIndexedOnZero * 4 + 3] = region.height

            displayPoints.geometry.getAttribute('regionVector').needsUpdate = true
            // don't register click here or duplicate will be counted from the webhook updates
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

    // 0 means no region is selected, by default it's like that for the whole map
    const generateDefaultRegionVector = (size: number) => {
        return new Float32Array(size * 4).fill(0);
    }

    displayPoints.geometry.setAttribute('regionVector', new THREE.BufferAttribute(generateDefaultRegionVector(size), 4));

    const updateTilesAccordingToNewBindings = (bindings: Map<number, string>) => {
        if (bindings.size == 0) return
        const regionVectors = displayPoints.geometry.getAttribute('regionVector').array as Float32Array;
        bindings.forEach((countryCode, index) => {
            const arrayIdIndexedOnZero = index - 1
            const region = regions.get(countryCode);
            if (!region) return

            regionVectors[arrayIdIndexedOnZero * 4] = region.x;
            regionVectors[arrayIdIndexedOnZero * 4 + 1] = region.y;
            regionVectors[arrayIdIndexedOnZero * 4 + 2] = region.width;
            regionVectors[arrayIdIndexedOnZero * 4 + 3] = region.height;
        });
        displayPoints.geometry.getAttribute('regionVector').needsUpdate = true;
    }


    const tilesPerBatch = 10_000
    ownershipsGetter.getCurrentOwnershipsByBatch(
        tilesPerBatch,
        size,
        (ownerships) => {
            leaderboard.registerOwnerships(ownerships)
            leaderboard.commitUpdate() // recompute the leaderboard after each batch
            updateTilesAccordingToNewBindings(ownerships.bindings)
        },
        lifetime.signal,
    ).catch((e) => {
        if (lifetime.signal.aborted) return
        console.error("Failed to fetch initial ownerships", e)
    })


    const cleanUpdatesListener = updatesListener.listenForUpdatesBatch(
        (updates: Update[]) => {
            const bindings = new Map<number, string>()
            updates.forEach(u => {
                /**
                 * A code the server knows and we do not is skipped, not thrown
                 * on: throwing here used to drop the whole batch, and with it
                 * every other tile in the same frame.
                 */
                const country = Countries.get(u.newCountry)
                if (!country) {
                    warnOnce(`Ignoring an update for unknown country "${u.newCountry}"`)
                    return
                }

                const oldCountry = u.previousCountry ? Countries.get(u.previousCountry) : undefined

                leaderboard.registerClick(oldCountry, country)
                bindings.set(u.tile, u.newCountry)
            })
            updateTilesAccordingToNewBindings(bindings)
            leaderboard.commitUpdate() // commit after each batch
        })


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
