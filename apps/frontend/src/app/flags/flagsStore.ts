import {useEffect} from "react"
import {create} from "zustand"
import {FlagCatalog, FlagReads, ListedFlag} from "../../domain/flags.ts"
import {FlagLister, FlagStream} from "../../backends/flags.ts"
import {nameOpenedFlags} from "../../i18n/translator.ts"

type FlagsStore = {
    catalog: FlagCatalog
    // Whether a read ended since the store was served, answered or not.
    read: boolean
    // How many times the stream opened: what else changed while it was closed is read again then too.
    openings: number
    // None stops reading and forgets the list.
    serve: (lister: FlagLister | undefined) => void
    // The list may have changed unheard: the stream opened.
    changed: () => void
    // A flag opened, closed or renamed, as the stream tells it: no read.
    told: (flag: ListedFlag) => void
    missed: (code: string) => void
}

export const useFlags = create<FlagsStore>()((set, get) => {
    let lister: FlagLister | undefined
    let served = 0
    let reads = new FlagReads()
    // What the stream told during the read in flight, which its answer may predate.
    let heard: ListedFlag[] = []

    const read = (fresh: boolean) => {
        const from = lister
        if (from === undefined) return
        const generation = served
        const current = () => generation === served
        heard = []
        from.listFlags(fresh).then(
            (listed) => {
                if (current()) set({catalog: FlagCatalog.BUILT_IN.withFlags(listed).withFlags(heard)})
            },
            (e: unknown) => console.error("could not read the list of flags: the page keeps the flags it knows", e),
        ).finally(() => {
            if (!current()) return
            if (!get().read) set({read: true})
            if (reads.ended()) read(true)
        })
    }

    return {
        catalog: FlagCatalog.BUILT_IN,
        read: false,
        openings: 0,
        serve: (next) => {
            lister = next
            served += 1
            reads = new FlagReads()
            heard = []
            set({catalog: FlagCatalog.BUILT_IN, read: false})
            if (next !== undefined && reads.changed()) read(false)
        },
        changed: () => {
            if (lister === undefined) return
            set(({openings}) => ({openings: openings + 1}))
            if (reads.changed()) read(true)
        },
        told: (flag) => {
            if (lister === undefined) return
            heard.push(flag)
            set({catalog: get().catalog.withFlags([flag])})
        },
        missed: (code) => {
            if (lister !== undefined && reads.missed(code)) read(true)
        },
    }
})

export function flagCatalog(): FlagCatalog {
    return useFlags.getState().catalog
}

// A flag the page does not know asks for the list again.
export function knowsFlag(code: string): boolean {
    const {catalog, missed} = useFlags.getState()
    if (catalog.knows(code)) return true
    missed(code)
    return false
}

nameOpenedFlags((code) => {
    const name = flagCatalog().openedName(code)
    if (name === undefined) useFlags.getState().missed(code)
    return name
})

export function afterFirstRead(then: (catalog: FlagCatalog) => void): void {
    if (useFlags.getState().read) {
        then(flagCatalog())
        return
    }
    const stop = useFlags.subscribe((store) => {
        if (!store.read) return
        stop()
        then(store.catalog)
    })
}

// Reads the list at load and each time the flags' stream opens, takes each flag it tells, and forgets them once it stops.
export function useFlagsFeed(flags: FlagLister & FlagStream): void {
    useEffect(() => {
        const {serve, changed, told} = useFlags.getState()
        serve(flags)
        const stop = flags.listenForFlags({onOpened: changed, onChanged: told})
        return () => {
            stop()
            serve(undefined)
        }
    }, [flags])
}
