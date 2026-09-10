// SCRATCH — R&D harness only, not for merge.
// Serves a frozen production map snapshot (`public/dev-map.bin`) so the globe can
// be looked at with realistic ownership from localhost, where CORS blocks the
// real API.
import {Ownerships, OwnershipsGetter, TileClicker, UpdatesListener} from "./backend.ts"

export class SnapshotBackend implements TileClicker, OwnershipsGetter, UpdatesListener {
    private owners: string[] = []
    private loading: Promise<void> | undefined

    // On first use rather than at module scope: awaiting this from the top level
    // of main.tsx builds under Vite's dev server and then fails the production
    // bundle, whose browser targets predate top-level await.
    private load(): Promise<void> {
        return this.loading ??= (async () => {
            const buffer = await (await fetch("/dev-map.bin")).arrayBuffer()
            const view = new DataView(buffer)
            const headerBytes = view.getUint32(0, true)
            const header = JSON.parse(new TextDecoder().decode(new Uint8Array(buffer, 4, headerBytes)))
            const tiles = new Uint16Array(buffer.slice(4 + headerBytes))
            this.owners = new Array(header.tiles + 1)
            for (let i = 0; i < header.tiles; i++) this.owners[i + 1] = header.codes[tiles[i]]
        })()
    }

    async clickTile(tile: number, country: string) {
        await this.load()
        this.owners[tile] = country
    }

    async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
    ) {
        await this.load()
        for (let start = 1; start <= maxIndex; start += batchSize) {
            const bindings = new Map<number, string>()
            for (let tile = start; tile < Math.min(start + batchSize, maxIndex + 1); tile++) {
                if (this.owners[tile]) bindings.set(tile, this.owners[tile])
            }
            callback({bindings})
        }
    }

    listenForUpdates(): () => void {
        return () => {
        }
    }

    listenForUpdatesBatch(): () => void {
        return () => {
        }
    }
}
