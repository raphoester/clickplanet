import type {ListedFlag} from "../domain/flags.ts"

export interface FlagLister {
    // fresh: past the browser's copy of the last answer, for a read after the list changed.
    listFlags(fresh: boolean): Promise<ListedFlag[]>
}

export type FlagHandlers = {
    // Each connection's first message: what changed while the stream was closed is only in a read of the list.
    onOpened: () => void
    // A flag opened, closed or renamed, as it is now.
    onChanged: (flag: ListedFlag) => void
    onDeleted: (code: string) => void
}

// The flags module's stream, open while it is listened to.
export interface FlagStream {
    listenForFlags(handlers: FlagHandlers): () => void
}
