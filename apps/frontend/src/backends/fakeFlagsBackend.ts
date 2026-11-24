import countriesData from "../../static/countries/countries.json"
import type {FlagPicture, ListedFlag, Pin} from "../domain/flags.ts"
import {FlagHandlers, FlagLister, FlagStream} from "./flags.ts"

const BUILT_IN: readonly ListedFlag[] = Object.entries(countriesData).map(([code, name]) => ({code, name, closed: false}))

const CODE_LETTERS = "abcdefghijklmnopqrstuvwxyz0123456789"
const DAY_MS = 24 * 60 * 60 * 1000
// Time added within this long of a close opens the flag again.
const COMEBACK_MS = 30 * DAY_MS
const MAX_NAME = 40

// A flag's picture, as the server would have drawn it from an upload.
export type FlagPainter = (name: string) => Promise<FlagPicture>

// What the server refuses of a flag's name: 1 to 40 characters, and no other flag's, built-in or sold, whatever its case.
export type NameRefusal = "empty" | "too long" | "taken"

// As the server lists the flags that left the build: sold, under the codes they shipped with.
const LEFT_THE_BUILD: readonly {code: string, name: string, hash: string, height: number, focusX: number, focusY: number, pin: Pin}[] = [
    {code: "xb", name: "Brittany", hash: "ed47508c67e67af2c0711725441736fbe62b614fb9b9c587a05efc51023c3eb6", height: 512,
        focusX: 0.4344, focusY: 0.5151, pin: {latitude: 48.1173, longitude: -1.6778}},
    {code: "gb-eng", name: "England", hash: "a5bbe1742db6aa08e5850c063ca9feb11b17c99fbf2546f07ed1f2e57bb44edb", height: 461,
        focusX: 0.5, focusY: 0.5, pin: {latitude: 51.501941, longitude: -0.118668}},
    {code: "gb-sct", name: "Scotland", hash: "e044e22d0d52a24b10503b26b85c66d0c9223e2ddd07842b65d0c9b00ab0a5f8", height: 461,
        focusX: 0.5, focusY: 0.5, pin: {latitude: 55.948328, longitude: -3.219091}},
    {code: "gb-wls", name: "Wales", hash: "625234995a5a9c1da4e98805b1f62053d49ee570cc82c79494ae5e6914fb32d5", height: 461,
        focusX: 0.5, focusY: 0.4711, pin: {latitude: 51.499995, longitude: -3.225008}},
]

// `openFlag("Bavaria")`, `addTime(code, days)`, `closeFlag(code, note)` and `renameFlag(code, name)` on the console, each told
// as the flags module's stream tells it. Nothing closes a flag when its time runs out: closeFlag does.
export class FakeFlagsBackend implements FlagLister, FlagStream {
    private readonly sold: ListedFlag[]
    private readonly listeners = new Set<FlagHandlers>()
    private readonly closings = new Set<(closed: ReadonlySet<string>) => void>()
    private readonly closeNotes = new Map<string, string>()

    constructor(private readonly paint: FlagPainter, private readonly now: () => number = Date.now) {
        this.sold = LEFT_THE_BUILD.map(({code, name, hash, height, focusX, focusY, pin}) => ({code, name, closed: false, sold: {
            picture: {url: `https://media.clickplanet.lol/flags/pictures/${hash}.png`, width: 768, height, stretch: false, focusX, focusY},
            endsAt: now() + 30 * DAY_MS,
            pin,
        }}))
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

    // A submission opened brings its own picture and pin; the console's openFlag has its picture painted from its name.
    public async open(name: string, days = 30, from: {picture?: FlagPicture, pin?: Pin} = {}): Promise<{code: string} | {refused: NameRefusal}> {
        const trimmed = name.trim()
        const refused = this.nameRefusalOf(trimmed, undefined)
        if (refused !== undefined) return {refused}
        const code = `r-${Array.from({length: 10}, () => CODE_LETTERS[Math.floor(Math.random() * CODE_LETTERS.length)]).join("")}`
        const picture = from.picture ?? await this.paint(trimmed)
        const flag: ListedFlag = {code, name: trimmed, closed: false, sold: {picture, endsAt: this.now() + days * DAY_MS, pin: from.pin}}
        this.sold.push(flag)
        this.tell(flag)
        return {code}
    }

    // A change approved: the flag's new name, and its new picture when one came.
    public change(code: string, name: string, picture?: FlagPicture): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined) return false
        flag.name = name
        if (picture) flag.sold = {...flag.sold, picture}
        this.tell(flag)
        return true
    }

    // As the stream would open again after a drop: the page reads what changed while it was closed.
    public reopen(): void {
        this.listeners.forEach(({onOpened}) => onOpened())
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
        this.closeNotes.delete(code)
        flag.sold = {picture: flag.sold.picture, endsAt: now + days * DAY_MS}
        this.tell(flag)
        return "reopened"
    }

    // The note its founder reads, as ListMySubmissions answers it.
    public close(code: string, note: string): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined || flag.closed || note.trim() === "") return false
        flag.closed = true
        this.closeNotes.set(code, note.trim())
        flag.sold = {...flag.sold, closedAt: this.now()}
        this.tell(flag)
        return true
    }

    public closeNoteOf(code: string): string | undefined {
        return this.closeNotes.get(code)
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
        const closed = this.closed()
        this.closings.forEach((onClosed) => onClosed(closed))
    }
}

function copyOf(flag: ListedFlag): ListedFlag {
    return flag.sold ? {...flag, sold: {...flag.sold}} : {...flag}
}
