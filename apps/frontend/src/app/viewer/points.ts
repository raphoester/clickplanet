import {COORDINATES_URL} from "./coordinatesAsset.ts"
import {LANDMASK_URL} from "./landmaskAsset.ts"
import {decodeCoordinates, type PointGeometryData} from "./coordinatesBinary.ts"
import type {LatticeAnswer} from "./latticeWorker.ts"
import {cellsOf, type TileCells} from "../../domain/latticeCells.ts"

export type TileGeometry = PointGeometryData & {cells: TileCells}

export async function loadPointGeometryData(signal?: AbortSignal): Promise<TileGeometry> {
    const landmask = await fetched(LANDMASK_URL, signal)
    try {
        const {positions, cells} = await rebuilt(landmask, signal)
        return {positions, size: positions.length / 3, cells}
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not rebuild the tile positions on ${navigator.userAgent}, downloading them`, e)
        const downloaded = decodeCoordinates(await fetched(COORDINATES_URL, signal))
        return {...downloaded, cells: cellsOf(downloaded.positions)}
    }
}

async function fetched(url: string, signal?: AbortSignal): Promise<ArrayBuffer> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`Failed to fetch ${url}: ${response.status} ${response.statusText}`)
    return response.arrayBuffer()
}

function rebuilt(landmask: ArrayBuffer, signal?: AbortSignal): Promise<{positions: Float32Array, cells: TileCells}> {
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
            if ("positions" in data) resolve(data)
            else reject(new Error(data.error))
        })
        worker.onerror = (event) => settle(() => reject(new Error(event.message || "the lattice worker did not start")))
        worker.postMessage(landmask, [landmask])
    })
}
