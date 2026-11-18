// Which board the game page draws, and the reload when the server plays another (docs/board-reload.md).
//
// A page plays one board for its life: the one GetMap names at its boot. A later answer naming another reloads it, as a
// restart does (restarts.ts), never a rebuild in place: the stream's opening at once, since a stream is never cached;
// a GetMap read only once a read made past the browser's cache still names it, since GetMap is a GET kept 5 s.

import {FIRST_BOARD} from "../domain/board.ts"

// GetMap's max-age: a read this long after another is not the browser's copy of it.
export const CACHED_MS = 5_000

// After a reload for a board, a boot that reads another reads GetMap again this often, up to SETTLE_MS: the browser
// may still hold its copy from before the reload.
const READ_AGAIN_MS = 1_000
const SETTLE_MS = 6_000

// A reload is remembered this long: a page loaded later is a page of its own, and may reload once again.
const REMEMBERED_MS = 60_000

const STORAGE_KEY = "clickplanet-board-reload"

// What a reload for a board leaves the page it reloads: the board that caused it, and when.
export type BoardReload = {board: string, at: number}

export type ReloadMemory = {
    read(): BoardReload | undefined
    write(reload: BoardReload): void
    forget(): void
}

// The tab's own: a reload keeps it, another tab does not see it, and a page with no storage reloads at most once a load.
export function sessionMemory(storage: () => Storage = () => window.sessionStorage): ReloadMemory {
    return {
        read: () => {
            try {
                const kept = JSON.parse(storage().getItem(STORAGE_KEY) ?? "null") as Partial<BoardReload> | null
                return typeof kept?.board === "string" && typeof kept.at === "number" ? {board: kept.board, at: kept.at} : undefined
            } catch {
                return undefined
            }
        },
        write: (reload) => {
            try {
                storage().setItem(STORAGE_KEY, JSON.stringify(reload))
            } catch {
                // No storage: the next load reads GetMap as a first one does.
            }
        },
        forget: () => {
            try {
                storage().removeItem(STORAGE_KEY)
            } catch {
                // No storage: nothing was kept.
            }
        },
    }
}

// A board this build does not carry, after the reload that would have fetched a newer build.
export class StaleBundleError extends Error {
    constructor(readonly board: string) {
        super(`the server plays board "${board}", which this page does not carry, even after a reload`)
        this.name = "StaleBundleError"
    }
}

// What the page does with a GetMap answer: draw it, read it again past the browser's cache, or leave, as it reloads.
export type Verdict = "drawn" | "readAgain" | "reloading"

export type BoardChangesOptions = {
    reload: () => void
    // Whether this build carries the board.
    carries: (board: string) => boolean
    memory?: ReloadMemory
    now?: () => number
    wait?: (ms: number, signal?: AbortSignal) => Promise<void>
}

export class BoardChanges {
    private readonly reload: () => void
    private readonly carries: (board: string) => boolean
    private readonly memory: ReloadMemory
    private readonly now: () => number
    private readonly wait: (ms: number, signal?: AbortSignal) => Promise<void>
    private drawn: string | undefined
    // The board the stream opened on before the page chose its own: never a cached answer.
    private streamed: string | undefined
    // When a GetMap read first named another board than the one drawn.
    private otherSince: number | undefined
    private reloading = false

    constructor(options: BoardChangesOptions) {
        this.reload = options.reload
        this.carries = options.carries
        this.memory = options.memory ?? sessionMemory()
        this.now = options.now ?? Date.now
        this.wait = options.wait ?? pause
    }

    get board(): string | undefined {
        return this.drawn
    }

    // The board the page draws, at its boot, from read (a GetMap). After a reload for a board, an answer naming another
    // is read again for a few seconds: it may be the browser's copy from before the reload. A board this build does not
    // carry reloads the page once, for a newer build, then fails. While the page reloads, this never settles.
    async choose(read: (signal?: AbortSignal) => Promise<string>, signal?: AbortSignal): Promise<string> {
        const kept = this.memory.read()
        const remembered = kept !== undefined && this.now() - kept.at < REMEMBERED_MS ? kept : undefined
        let board = named(await read(signal))
        if (remembered !== undefined && board !== remembered.board) {
            const until = this.now() + SETTLE_MS
            while (board !== remembered.board && this.streamed === undefined && this.now() < until) {
                await this.wait(READ_AGAIN_MS, signal)
                board = named(await read(signal))
            }
        }
        board = this.streamed ?? board

        if (!this.carries(board)) {
            if (remembered?.board === board) throw new StaleBundleError(board)
            this.leaveFor(board)
            return new Promise<never>(() => {})
        }
        this.memory.forget()
        this.drawn = board
        return board
    }

    // The stream opened on board: never the browser's copy, so another than the one drawn reloads the page at once.
    // Whether the page reloads.
    opened(board: string): boolean {
        if (board === "") return this.reloading
        if (this.drawn === undefined) {
            this.streamed = board
            return false
        }
        if (board !== this.drawn) this.leaveFor(board)
        return this.reloading
    }

    // A GetMap answer named board. Another than the one drawn is read again once the browser's copy is past: the page
    // reloads when that read still names it.
    answered(board: string): Verdict {
        if (this.reloading) return "reloading"
        const read = named(board)
        if (this.drawn === undefined || read === this.drawn) {
            this.otherSince = undefined
            return "drawn"
        }
        const now = this.now()
        if (this.otherSince === undefined) {
            this.otherSince = now
            return "readAgain"
        }
        if (now - this.otherSince < CACHED_MS) return "readAgain"
        this.leaveFor(read)
        return "reloading"
    }

    private leaveFor(board: string) {
        if (this.reloading) return
        this.reloading = true
        this.memory.write({board, at: this.now()})
        this.reload()
    }
}

// An answer from a server before boards were named names none: its map was on the first board.
function named(board: string): string {
    return board === "" ? FIRST_BOARD : board
}

function pause(ms: number, signal?: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
        signal?.throwIfAborted()
        const timer = setTimeout(resolve, ms)
        signal?.addEventListener("abort", () => {
            clearTimeout(timer)
            reject(signal.reason)
        }, {once: true})
    })
}
