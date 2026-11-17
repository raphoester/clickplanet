import {BOARD} from "./boardAsset.ts"
import type {LatticeAnswer, LatticeQuestion} from "./latticeWorker.ts"
import {rebuildTiles, type RebuiltTiles} from "../../domain/board.ts"
import type {TileCells} from "../../domain/latticeCells.ts"
import type {TileChunks} from "../../domain/tileChunks.ts"

export type TileGeometry = {positions: Float32Array, size: number, cells: TileCells, chunks: TileChunks}

export async function loadPointGeometryData(signal?: AbortSignal): Promise<TileGeometry> {
    const question: LatticeQuestion = {lattice: BOARD.lattice, tiles: BOARD.tiles, runs: await fetched(BOARD.urls.runs, signal)}
    let rebuilt: RebuiltTiles
    try {
        rebuilt = await inWorker(question, signal)
    } catch (e) {
        if (signal?.aborted) throw e
        console.error(`could not rebuild the tile positions in a worker on ${navigator.userAgent}, rebuilding them on the page`, e)
        rebuilt = rebuildTiles(question.lattice, question.tiles, question.runs)
    }
    if (rebuilt.hash !== BOARD.positionsHash) {
        console.error(`the tile positions rebuilt on ${navigator.userAgent} hash to ${hex(rebuilt.hash)}, `
            + `${BOARD.name} says ${hex(BOARD.positionsHash)}`)
    }
    return {positions: rebuilt.positions, size: rebuilt.positions.length / 3, cells: rebuilt.cells, chunks: rebuilt.chunks}
}

async function fetched(url: string, signal?: AbortSignal): Promise<ArrayBuffer> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`Failed to fetch ${url}: ${response.status} ${response.statusText}`)
    return response.arrayBuffer()
}

// Copied to the worker, not moved: the page rebuilds from the same runs if the worker fails.
function inWorker(question: LatticeQuestion, signal?: AbortSignal): Promise<RebuiltTiles> {
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
        worker.postMessage(question)
    })
}

function hex(hash: number): string {
    return hash.toString(16).padStart(8, "0")
}
