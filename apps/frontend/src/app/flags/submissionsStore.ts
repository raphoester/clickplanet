import {useEffect} from "react"
import {create} from "zustand"
import {FlagSubmitter} from "../../backends/flags.ts"
import {FlagCatalog} from "../../domain/flags.ts"
import {Held, Submission, untold, withTold} from "../../domain/flagSubmissions.ts"
import {useNews} from "../news/newsStore.ts"
import {useFlags} from "./flagsStore.ts"
import {keepTold, toldSubmissions} from "./toldSubmissions.ts"

// A signed-in account submits; a guest offered sign-in is shown the buttons; nobody else sees the entry.
export type Who = "nobody" | "guest" | "player"

type FlagSubmissionsStore = {
    who: Who
    // The account's, the newest first: none for a guest.
    submissions: readonly Submission[]
    // The flags the account holds now, each removed one with its note.
    held: readonly Held[]
    // Whether a read ended since the store was served.
    read: boolean
    setWho: (who: Who) => void
    // None forgets the list.
    serve: (submitter: FlagSubmitter | undefined) => void
    refresh: () => void
    // The answer of a send, before the next read says it, in the place of the one it replaced.
    submitted: (submission: Submission, replaced?: string) => void
    // The answer of an appeal, before the next read says it, in its place: none once signed out, as none is held.
    changed: (submission: Submission) => void
}

export const useFlagSubmissions = create<FlagSubmissionsStore>()((set, get) => {
    let submitter: FlagSubmitter | undefined
    let served = 0
    let reading = false
    let queued = false

    const read = () => {
        const from = submitter
        if (from === undefined) return
        if (reading) {
            queued = true
            return
        }
        reading = true
        const generation = served
        const current = () => generation === served
        from.myFlags().then(
            ({submissions, held}) => {
                if (current()) set({submissions, held, read: true})
            },
            (e: unknown) => console.error("could not read the flags you submitted", e),
        ).finally(() => {
            if (!current()) return
            reading = false
            if (!get().read) set({read: true})
            if (queued) {
                queued = false
                read()
            }
        })
    }

    return {
        who: "nobody",
        submissions: [],
        held: [],
        read: false,
        setWho: (who) => set({who}),
        serve: (next) => {
            submitter = next
            served += 1
            reading = false
            queued = false
            set({submissions: [], held: [], read: false})
            read()
        },
        refresh: read,
        submitted: (submission, replaced) => {
            if (submitter === undefined) return
            set(({submissions}) => ({
                submissions: [submission, ...submissions.filter(({id}) => id !== submission.id && id !== replaced)],
            }))
        },
        changed: (submission) => {
            set(({submissions}) => ({submissions: submissions.map((kept) => kept.id === submission.id ? submission : kept)}))
        },
    }
})

export const VERDICT_READ_EVERY_MS = 3_000
export const VERDICT_READ_FOR_MS = 120_000

// The judge reviews a flag within seconds of its send, paid or not: while the paywall shows one that waits, its verdict
// is read again, for a while.
export function useVerdictReads(waiting: boolean, now: () => number = Date.now): void {
    useEffect(() => {
        if (!waiting) return
        const until = now() + VERDICT_READ_FOR_MS
        const timer = setInterval(() => {
            if (now() >= until) {
                clearInterval(timer)
                return
            }
            useFlagSubmissions.getState().refresh()
        }, VERDICT_READ_EVERY_MS)
        return () => clearInterval(timer)
    }, [waiting, now])
}

// A signed-in account's submissions: read at once, each time the flags stream opens, and when the form opens; each
// decision the page sees first is told once on the news line.
export function useFlagSubmissionsFeed(submitter: FlagSubmitter, who: Who): void {
    const signedIn = who === "player"

    useEffect(() => {
        const {setWho} = useFlagSubmissions.getState()
        setWho(who)
        return () => setWho("nobody")
    }, [who])

    useEffect(() => {
        if (!signedIn) return
        const {serve, refresh} = useFlagSubmissions.getState()
        serve(submitter)
        const stopOpenings = useFlags.subscribe((flags, was) => {
            if (flags.openings !== was.openings || heldChanged(flags.catalog, was.catalog)) refresh()
        })
        return () => {
            stopOpenings()
            serve(undefined)
        }
    }, [submitter, signedIn])

    useEffect(() => {
        const stopSubmissions = useFlagSubmissions.subscribe(({submissions, held}, was) => {
            if (submissions !== was.submissions || held !== was.held) tellDecisions()
        })
        const stopFlags = useFlags.subscribe(({catalog, read}, was) => {
            if (catalog !== was.catalog || read !== was.read) tellDecisions()
        })
        return () => {
            stopSubmissions()
            stopFlags()
        }
    }, [])
}

// A flag held that closes, comes back or opens for another player: the removal's note and who holds it are read with
// the submissions.
function heldChanged(catalog: FlagCatalog, was: FlagCatalog): boolean {
    if (catalog === was) return false
    return useFlagSubmissions.getState().held
        .some(({code}) => catalog.closedFlag(code) !== was.closedFlag(code))
}

// Not before the list of flags is read: an opened flag would be told open, then closed.
function tellDecisions(): void {
    const {catalog, read} = useFlags.getState()
    if (!read) return
    const told = toldSubmissions()
    if (told === undefined) return
    const {submissions, held} = useFlagSubmissions.getState()
    const fresh = untold(submissions, told, catalog, held)
    if (fresh.length === 0) return
    keepTold(withTold(told, fresh))
    useNews.getState().tellSubmission(fresh[0])
}
