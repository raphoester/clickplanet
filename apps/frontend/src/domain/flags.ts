import {Countries} from "./countries.ts"

export type ListedFlag = {code: string, name: string, closed: boolean}

const OPENED_CODE = /^r-[a-z0-9]{10}$/

export function isOpenedCode(code: string): boolean {
    return OPENED_CODE.test(code)
}

// A closed flag stays known, since history names it, but no player may play for it.
export class FlagCatalog {
    static readonly BUILT_IN = new FlagCatalog(new Map(), new Set())

    private constructor(
        private readonly opened: ReadonlyMap<string, string>,
        private readonly closed: ReadonlySet<string>,
    ) {
    }

    // A flag the game ships with keeps the page's own names.
    withList(listed: readonly ListedFlag[]): FlagCatalog {
        const opened = new Map<string, string>()
        const closed = new Set<string>()
        for (const {code, name, closed: shut} of listed) {
            if (!Countries.has(code)) opened.set(code, name)
            if (shut) closed.add(code)
        }
        return new FlagCatalog(opened, closed)
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
