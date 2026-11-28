import {Countries} from "./countries.ts"

// At most 768 x 512, between square and 2 wide by 1 high; stretch and focus are what flagFit.json says of a built flag.
export type FlagPicture = {url: string, width: number, height: number, stretch: boolean, focusX: number, focusY: number}

// A point on the globe, in degrees: never a tile, since each board numbers its tiles its own way.
export type Pin = {latitude: number, longitude: number}

// A flag opened since the build: its picture, when its time runs out, in milliseconds since the epoch, and where it
// starts in Capital Siege. Closed at its end, any player may pay for it and hold it; removed by the operator, nobody.
// Its anthem is the code of the flag the game ships with whose recording it plays, when its holder chose one.
export type SoldFlag = {
    picture: FlagPicture, endsAt: number, closedAt?: number, pin?: Pin, removed?: boolean, anthem?: string
}

export type ListedFlag = {code: string, name: string, closed: boolean, sold?: SoldFlag}

// Two letters or gb- and three, as the build's, or r- and ten drawn for a flag opened since.
const FLAG_CODE = /^([a-z]{2}(-[a-z]{3})?|r-[a-z0-9]{10})$/

// A flag the build does not ship: one opened since, or one sold since it left the build (Brittany, xb).
export function isOpenedCode(code: string): boolean {
    return FLAG_CODE.test(code) && !Countries.has(code)
}

// A closed flag stays known, since history names it, but no player may play for it.
export class FlagCatalog {
    static readonly BUILT_IN = new FlagCatalog(new Map(), new Set(), new Map())

    private constructor(
        private readonly opened: ReadonlyMap<string, string>,
        private readonly closed: ReadonlySet<string>,
        private readonly sold: ReadonlyMap<string, SoldFlag>,
    ) {
    }

    // Each flag as it is now, in order, over the catalog. A flag the game ships with keeps the page's own names.
    withFlags(flags: readonly ListedFlag[]): FlagCatalog {
        const opened = new Map(this.opened)
        const closed = new Set(this.closed)
        const sold = new Map(this.sold)
        for (const flag of flags) {
            if (!Countries.has(flag.code)) opened.set(flag.code, flag.name)
            if (flag.closed) closed.add(flag.code)
            else closed.delete(flag.code)
            if (flag.sold && !Countries.has(flag.code)) sold.set(flag.code, flag.sold)
        }
        return new FlagCatalog(opened, closed, sold)
    }

    knows(code: string): boolean {
        return Countries.has(code) || this.opened.has(code)
    }

    playable(code: string): boolean {
        return this.knows(code) && !this.closed.has(code)
    }

    playableCodes(): string[] {
        return [...Countries.keys(), ...this.opened.keys()].filter((code) => !this.closed.has(code))
    }

    openedName(code: string): string | undefined {
        return this.opened.get(code)
    }

    closedFlag(code: string): boolean {
        return this.closed.has(code)
    }

    // The code whose recording plays as the flag's anthem: a flag the game ships with plays its own, a sold one the one
    // its holder chose, or none.
    recordingOf(code: string): string | undefined {
        return Countries.has(code) ? code : this.sold.get(code)?.anthem
    }

    pictureOf(code: string): FlagPicture | undefined {
        return this.sold.get(code)?.picture
    }

    soldOf(code: string): SoldFlag | undefined {
        return this.sold.get(code)
    }

    // Closed at its end, not removed: any signed-in player may pay for it, and so hold it.
    claimable(code: string): boolean {
        return this.closed.has(code) && this.sold.has(code) && !this.sold.get(code)?.removed
    }

    claimableCodes(): string[] {
        return [...this.sold.keys()].filter((code) => this.claimable(code))
    }

    // The claimable flag of that name, whatever its case: a place has one flag.
    claimableNamed(name: string): string | undefined {
        const lower = name.trim().toLowerCase()
        if (lower === "") return undefined
        return this.claimableCodes().find((code) => this.opened.get(code)?.toLowerCase() === lower)
    }

    // Every flag sold, closed ones too: a closed flag still holds land.
    pictures(): ReadonlyMap<string, FlagPicture> {
        return new Map([...this.sold].map(([code, {picture}]) => [code, picture]))
    }
}

// One read at a time, one queued behind it on a change, and one per unknown code: a code still unknown never loops.
export class FlagReads {
    private reading = false
    private queued = false
    private readonly asked = new Set<string>()

    // A read in flight answers for the code.
    missed(code: string): boolean {
        if (this.asked.has(code)) return false
        this.asked.add(code)
        return this.start()
    }

    // A read in flight may have been answered before the change.
    changed(): boolean {
        if (this.reading) {
            this.queued = true
            return false
        }
        return this.start()
    }

    ended(): boolean {
        this.reading = false
        if (!this.queued) return false
        this.queued = false
        return this.start()
    }

    private start(): boolean {
        if (this.reading) return false
        this.reading = true
        return true
    }
}
