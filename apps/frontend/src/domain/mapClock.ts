import type {MapVersion} from "../backends/backend.ts"

// The server start the client's versions are counted in, and the version the stream last opened at.
export class MapClock {
    private epoch: number | undefined
    private opening: MapVersion | undefined

    // A version of another start compares with nothing the client holds.
    public counts(version: MapVersion): boolean {
        this.epoch ??= version.epoch
        return version.epoch === this.epoch
    }

    // Whether the server started again since the versions the client holds.
    public opened(version: MapVersion): boolean {
        const restarted = this.epoch !== undefined && this.epoch !== version.epoch
        this.epoch = version.epoch
        this.opening = version
        return restarted
    }

    public get open(): boolean {
        return this.opening !== undefined
    }

    // A read older than the stream's opening may miss what happened in between: the stream never sends it.
    public current(read: MapVersion): boolean {
        return this.opening !== undefined && read.epoch === this.opening.epoch && read.seq >= this.opening.seq
    }
}
