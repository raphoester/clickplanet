import {useEffect} from "react"
import {create} from "zustand"
import {FlagCatalog, FlagReads} from "../../domain/flags.ts"
import {FlagChanges, FlagLister} from "../../backends/flags.ts"
import {nameOpenedFlags} from "../../i18n/translator.ts"

type FlagsStore = {
    catalog: FlagCatalog
    // Whether a read ended since the store was served, answered or not.
    read: boolean
    // None stops reading and forgets the list.
    serve: (lister: FlagLister | undefined) => void
    changed: () => void
    missed: (code: string) => void
}

export const useFlags = create<FlagsStore>()((set, get) => {
    let lister: FlagLister | undefined
    let served = 0
    let reads = new FlagReads()

    const read = (fresh: boolean) => {
        const from = lister
        if (from === undefined) return
        const generation = served
        const current = () => generation === served
        from.listFlags(fresh).then(
            (listed) => {
                if (current()) set({catalog: FlagCatalog.BUILT_IN.withList(listed)})
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
        serve: (next) => {
            lister = next
            served += 1
            reads = new FlagReads()
            set({catalog: FlagCatalog.BUILT_IN, read: false})
            if (next !== undefined && reads.changed()) read(false)
        },
        changed: () => {
            if (lister !== undefined && reads.changed()) read(true)
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

// Reads the list at load and on each change planet tells, and forgets it once it stops.
export function useFlagsFeed(lister: FlagLister, changes: FlagChanges): void {
    useEffect(() => {
        const {serve, changed} = useFlags.getState()
        serve(lister)
        const stop = changes.listenForFlagChanges(changed)
        return () => {
            stop()
            serve(undefined)
        }
    }, [lister, changes])
}
