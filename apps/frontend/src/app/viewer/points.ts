import {COORDINATES_URL} from "./coordinatesAsset.ts"
import {LANDMASK_URL} from "./landmaskAsset.ts"
import {decodeCoordinates, type PointGeometryData} from "./coordinatesBinary.ts"
import type {LatticeAnswer} from "./latticeWorker.ts"

export type {PointGeometryData}

export async function loadPointGeometryData(signal?: AbortSignal): Promise<PointGeometryData> {
    const landmask = await fetched(LANDMASK_URL, signal)
    try {
        const positions = await rebuilt(landmask, signal)
        return {positions, size: positions.length / 3}
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not rebuild the tile positions on ${navigator.userAgent}, downloading them`, e)
        return decodeCoordinates(await fetched(COORDINATES_URL, signal))
    }
}

async function fetched(url: string, signal?: AbortSignal): Promise<ArrayBuffer> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`Failed to fetch ${url}: ${response.status} ${response.statusText}`)
    return response.arrayBuffer()
}

function rebuilt(landmask: ArrayBuffer, signal?: AbortSignal): Promise<Float32Array> {
    return new Promise((resolve, reject) => {
        signal?.throwIfAborted()
        const worker = new Worker(new URL("./latticeWorker.ts", import.meta.url), {type: "module"})
        const settle = (finish: () => void) => {
            worker.terminate()
            signal?.removeEventListener("abort", abort)
            finish()
        }
        const abort = () => settle(() => reject(signal?.reason))
        signal?.addEventListener("abort", abort, {once: true})
        worker.onmessage = ({data}: MessageEvent<LatticeAnswer>) => settle(() => {
            if ("positions" in data) resolve(data.positions)
            else reject(new Error(data.error))
        })
        worker.onerror = (event) => settle(() => reject(new Error(event.message || "the lattice worker did not start")))
        worker.postMessage(landmask, [landmask])
    })
}
