/**
 * How many times a browser is told why its click emptied a tile rather than
 * taking it. Native land takes two clicks, and a newcomer clicking a native tile
 * abroad sees it go blank: that has to be said, or it reads as a click that went
 * wrong. Three times is enough to learn it; after that the dust on the tile says
 * the rest.
 */
export const CLEAR_NOTES = 3

export const CLEAR_NOTES_KEY = "clickplanet-home-soil-notes"

type Store = Pick<Storage, "getItem" | "setItem">

/**
 * The count of notes shown, kept in local storage. A storage that is missing or
 * throws (a private window) counts in memory instead, so the note is still said
 * no more than three times a page load.
 */
export class ClearNotes {
    private shown: number

    constructor(private readonly store: Store | undefined, private readonly limit = CLEAR_NOTES) {
        this.shown = readCount(store)
    }

    /** Whether the next clear is worth a note. */
    get due(): boolean {
        return this.shown < this.limit
    }

    /** Counts one note shown. */
    record(): void {
        this.shown++
        try {
            this.store?.setItem(CLEAR_NOTES_KEY, String(this.shown))
        } catch {
            // Only a convenience: the count in memory still holds for this page.
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
