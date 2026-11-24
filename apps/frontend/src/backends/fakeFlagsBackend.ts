import countriesData from "../../static/countries/countries.json"
import type {FlagPicture, ListedFlag} from "../domain/flags.ts"
import {FlagHandlers, FlagLister, FlagStream} from "./flags.ts"

const BUILT_IN: readonly ListedFlag[] = Object.entries(countriesData).map(([code, name]) => ({code, name, closed: false}))

const CODE_LETTERS = "abcdefghijklmnopqrstuvwxyz0123456789"
const MINUTE_MS = 60 * 1000
const DAY_MS = 24 * 60 * MINUTE_MS
// Time added within this long of a close opens the flag again.
const COMEBACK_MS = 30 * DAY_MS
const MAX_NAME = 40

// A flag's picture, as the server would have drawn it from an upload.
export type FlagPainter = (name: string) => Promise<FlagPicture>

// What the server refuses of a flag's name: 1 to 40 characters, and no other flag's, built-in or sold, whatever its case.
export type NameRefusal = "empty" | "too long" | "taken"

// `openFlag("Bavaria")`, `addTime(code, days)`, `closeFlag(code)`, `renameFlag(code, name)` and `deleteFlag(code)` on the
// console, each told as the flags module's stream tells it. Nothing closes a flag when its time runs out: closeFlag does.
export class FakeFlagsBackend implements FlagLister, FlagStream {
    private sold: ListedFlag[] = []
    private readonly played = new Set<string>()
    private readonly listeners = new Set<FlagHandlers>()
    private readonly closings = new Set<(closed: ReadonlySet<string>) => void>()

    constructor(private readonly paint: FlagPainter, private readonly now: () => number = Date.now) {
    }

    public async listFlags(): Promise<ListedFlag[]> {
        return [...BUILT_IN, ...this.sold.map(copyOf)]
    }

    public listenForFlags(handlers: FlagHandlers): () => void {
        this.listeners.add(handlers)
        // As the stream's first heartbeat, a moment after it opens.
        const heartbeat = setTimeout(() => handlers.onOpened(), 0)
        return () => {
            clearTimeout(heartbeat)
            this.listeners.delete(handlers)
        }
    }

    // The flags closed now, as planet hears them: the fake planet answers GetAllegiance with them.
    public listenForClosed(onClosed: (closed: ReadonlySet<string>) => void): () => void {
        this.closings.add(onClosed)
        onClosed(this.closed())
        return () => this.closings.delete(onClosed)
    }

    public async open(name: string, days = 30): Promise<{code: string} | {refused: NameRefusal}> {
        const trimmed = name.trim()
        const refused = this.nameRefusalOf(trimmed, undefined)
        if (refused !== undefined) return {refused}
        const code = `r-${Array.from({length: 10}, () => CODE_LETTERS[Math.floor(Math.random() * CODE_LETTERS.length)]).join("")}`
        const flag: ListedFlag = {code, name: trimmed, closed: false, sold: {picture: await this.paint(trimmed), endsAt: this.now() + days * DAY_MS}}
        this.sold.push(flag)
        this.tell(flag)
        return {code}
    }

    // An open flag ends later; one closed less than 30 days ago opens again, ending that many days from now.
    public addTime(code: string, days: number): "longer" | "reopened" | "unknown" | "too late" {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined) return "unknown"
        const now = this.now()
        if (!flag.closed) {
            flag.sold = {...flag.sold, endsAt: flag.sold.endsAt + days * DAY_MS}
            this.tell(flag)
            return "longer"
        }
        if (now - (flag.sold.closedAt ?? 0) > COMEBACK_MS) return "too late"
        flag.closed = false
        flag.sold = {picture: flag.sold.picture, endsAt: now + days * DAY_MS}
        this.tell(flag)
        return "reopened"
    }

    public close(code: string): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined || flag.closed) return false
        flag.closed = true
        flag.sold = {...flag.sold, closedAt: this.now()}
        this.tell(flag)
        return true
    }

    // Open or closed: its code, picture and land stay.
    public rename(code: string, name: string): "renamed" | "unknown" | NameRefusal {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag === undefined) return "unknown"
        const trimmed = name.trim()
        const refused = this.nameRefusalOf(trimmed, code)
        if (refused !== undefined) return refused
        flag.name = trimmed
        this.tell(flag)
        return "renamed"
    }

    // A pledge or a click for the flag, as planet hears it: such a flag is never deleted.
    public markPlayed(code: string): void {
        this.played.add(code)
    }

    // Only a closed flag nobody played for, closed a minute ago or more.
    public delete(code: string): "deleted" | "unknown" | "open" | "played" | "too soon" {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined) return "unknown"
        if (!flag.closed) return "open"
        if (this.played.has(code)) return "played"
        if (this.now() - (flag.sold.closedAt ?? 0) < MINUTE_MS) return "too soon"
        this.sold = this.sold.filter((sold) => sold.code !== code)
        this.listeners.forEach(({onDeleted}) => onDeleted(code))
        this.tellClosings()
        return "deleted"
    }

    private nameRefusalOf(name: string, renamed: string | undefined): NameRefusal | undefined {
        if (name === "") return "empty"
        if ([...name].length > MAX_NAME) return "too long"
        const lower = name.toLowerCase()
        const taken = [...BUILT_IN, ...this.sold].some((flag) => flag.code !== renamed && flag.name.toLowerCase() === lower)
        return taken ? "taken" : undefined
    }

    private closed(): ReadonlySet<string> {
        return new Set(this.sold.filter((flag) => flag.closed).map((flag) => flag.code))
    }

    private tell(flag: ListedFlag) {
        this.listeners.forEach(({onChanged}) => onChanged(copyOf(flag)))
        this.tellClosings()
    }

    private tellClosings() {
        const closed = this.closed()
        this.closings.forEach((onClosed) => onClosed(closed))
    }
}

function copyOf(flag: ListedFlag): ListedFlag {
    return flag.sold ? {...flag, sold: {...flag.sold}} : {...flag}
}
