export const CLEAR_NOTES = 3

export const CLEAR_NOTES_KEY = "clickplanet-home-soil-notes"

type Store = Pick<Storage, "getItem" | "setItem">

export class ClearNotes {
    private shown: number

    constructor(private readonly store: Store | undefined, private readonly limit = CLEAR_NOTES) {
        this.shown = readCount(store)
    }

    get due(): boolean {
        return this.shown < this.limit
    }

    record(): void {
        this.shown++
        try {
            this.store?.setItem(CLEAR_NOTES_KEY, String(this.shown))
        } catch {
            // storage unavailable
        }
    }
}

function readCount(store: Store | undefined): number {
    try {
        const count = Number.parseInt(store?.getItem(CLEAR_NOTES_KEY) ?? "", 10)
        return Number.isFinite(count) && count > 0 ? count : 0
    } catch {
        return 0
    }
}
