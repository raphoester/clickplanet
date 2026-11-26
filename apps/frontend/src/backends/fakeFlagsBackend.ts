import countriesData from "../../static/countries/countries.json"
import type {FlagPicture, ListedFlag, Pin} from "../domain/flags.ts"
import {FlagHandlers, FlagLister, FlagStream} from "./flags.ts"

const BUILT_IN: readonly ListedFlag[] = Object.entries(countriesData).map(([code, name]) => ({code, name, closed: false}))

const CODE_LETTERS = "abcdefghijklmnopqrstuvwxyz0123456789"
const DAY_MS = 24 * 60 * 60 * 1000
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

// What a payment for a flag did, or why it bought nothing.
export type Paid = "longer" | "reopened" | "claimed" | "unknown" | "removed" | "held"

// `openFlag("Bavaria")`, `addTime(code, days)`, `endFlag(code)`, `claimElsewhere(code)`, `removeFlag(code, note)` and
// `renameFlag(code, name)` on the console, each told as the flags module's stream tells it. Nothing ends a flag when its
// time runs out: endFlag does.
export class FakeFlagsBackend implements FlagLister, FlagStream {
    private readonly sold: ListedFlag[]
    private readonly listeners = new Set<FlagHandlers>()
    private readonly closings = new Set<(closed: ReadonlySet<string>) => void>()
    private readonly removalNotes = new Map<string, string>()
    // The flags the page's account holds: its payment opened them or claimed them last.
    private readonly held = new Set<string>()

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

    // A submission opened brings its own picture and pin, and is held by its founder; the console's openFlag has its
    // picture painted from its name, and no holder.
    public async open(
        name: string, days = 30, from: {picture?: FlagPicture, pin?: Pin, held?: boolean} = {},
    ): Promise<{code: string} | {refused: NameRefusal}> {
        const trimmed = name.trim()
        const refused = this.nameRefusalOf(trimmed, undefined)
        if (refused !== undefined) return {refused}
        const code = `r-${Array.from({length: 10}, () => CODE_LETTERS[Math.floor(Math.random() * CODE_LETTERS.length)]).join("")}`
        const picture = from.picture ?? await this.paint(trimmed)
        const flag: ListedFlag = {code, name: trimmed, closed: false, sold: {picture, endsAt: this.now() + days * DAY_MS, pin: from.pin}}
        this.sold.push(flag)
        if (from.held) this.held.add(code)
        this.tell(flag)
        return {code}
    }

    // The page's account's flags, each with the note of its removal.
    public heldFlags(): {code: string, removalNote?: string}[] {
        return [...this.held].map((code) => {
            const removalNote = this.removalNotes.get(code)
            return removalNote === undefined ? {code} : {code, removalNote}
        })
    }

    public holds(code: string): boolean {
        return this.held.has(code)
    }

    // As a payment of the page's account: its own flag ends later, or opens again; a flag whose time ended opens for it.
    public pay(code: string, days: number): Paid {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined) return "unknown"
        if (flag.sold.removed) return "removed"
        const holds = this.held.has(code)
        if (!flag.closed && !holds) return "held"
        const paid = this.timeAdded(flag, days)
        if (holds) return paid
        this.held.add(code)
        return "claimed"
    }

    // As the stream would open again after a drop: the page reads what changed while it was closed.
    public reopen(): void {
        this.listeners.forEach(({onOpened}) => onOpened())
    }

    // The operator's: an open flag ends later; a closed or removed one opens again, ending that many days from now, its
    // holder kept; a removed one only while no other flag took its name.
    public addTime(code: string, days: number): "longer" | "reopened" | "unknown" | "taken" {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined) return "unknown"
        if (flag.sold.removed && this.nameRefusalOf(flag.name, code) === "taken") return "taken"
        return this.timeAdded(flag, days)
    }

    // Its time ends now, as at its end: any player may pay for it and hold it.
    public end(code: string): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined || flag.closed) return false
        flag.closed = true
        flag.sold = {...flag.sold, endsAt: this.now(), closedAt: this.now()}
        this.tell(flag)
        return true
    }

    // Another player pays for a flag whose time ended: it opens again for them, and leaves the page's account's flags.
    public claimElsewhere(code: string, days = 30): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined || !flag.closed || flag.sold.removed) return false
        this.held.delete(code)
        this.timeAdded(flag, days)
        return true
    }

    // The operator's removal, open or closed at its end: for good, with the note its holder reads.
    public remove(code: string, note: string): boolean {
        const flag = this.sold.find((sold) => sold.code === code)
        if (flag?.sold === undefined || flag.sold.removed || note.trim() === "") return false
        flag.closed = true
        this.removalNotes.set(code, note.trim())
        flag.sold = {...flag.sold, closedAt: this.now(), removed: true}
        this.tell(flag)
        return true
    }

    private timeAdded(flag: ListedFlag, days: number): "longer" | "reopened" {
        if (flag.sold === undefined) throw new Error(`${flag.code} is no sold flag`)
        if (!flag.closed) {
            flag.sold = {...flag.sold, endsAt: flag.sold.endsAt + days * DAY_MS}
            this.tell(flag)
            return "longer"
        }
        flag.closed = false
        this.removalNotes.delete(flag.code)
        flag.sold = {picture: flag.sold.picture, endsAt: this.now() + days * DAY_MS, pin: flag.sold.pin}
        this.tell(flag)
        return "reopened"
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
        const taken = [...BUILT_IN, ...this.sold]
            .some((flag) => flag.code !== renamed && !flag.sold?.removed && flag.name.toLowerCase() === lower)
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
