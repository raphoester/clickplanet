import type {ListedFlag} from "../domain/flags.ts"

export interface FlagLister {
    // fresh: past the browser's copy of the last answer, for a read after the list changed.
    listFlags(fresh: boolean): Promise<ListedFlag[]>
}

// Planet's stream: a flag opened or closed, or a gap in which one may have been.
export interface FlagChanges {
    listenForFlagChanges(onChanged: () => void): () => void
}
