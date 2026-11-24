import {useEffect} from "react"
import {create} from "zustand"
import {FlagSubmitter} from "../../backends/flags.ts"
import {latestOf, Submission, untold, withTold} from "../../domain/flagSubmissions.ts"
import {useNews} from "../news/newsStore.ts"
import {useFlags} from "./flagsStore.ts"
import {keepTold, toldSubmissions} from "./toldSubmissions.ts"

// A signed-in account submits; a guest offered sign-in is shown the buttons; nobody else sees the entry.
export type Who = "nobody" | "guest" | "player"

type FlagSubmissionsStore = {
    who: Who
    // The account's, the newest first: none for a guest.
    submissions: readonly Submission[]
    // Whether a read ended since the store was served.
    read: boolean
    setWho: (who: Who) => void
    // None forgets the list.
    serve: (submitter: FlagSubmitter | undefined) => void
    refresh: () => void
    // The answer of a send, before the next read says it.
    submitted: (submission: Submission) => void
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
        from.mySubmissions().then(
            (submissions) => {
                if (current()) set({submissions, read: true})
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
        read: false,
        setWho: (who) => set({who}),
        serve: (next) => {
            submitter = next
            served += 1
            reading = false
            queued = false
            set({submissions: [], read: false})
            read()
        },
        refresh: read,
        submitted: (submission) => {
            if (submitter === undefined) return
            set(({submissions}) => ({submissions: [submission, ...submissions.filter(({id}) => id !== submission.id)]}))
        },
    }
})

export const latestSubmission = (store: FlagSubmissionsStore): Submission | undefined => latestOf(store.submissions)

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
            if (flags.openings !== was.openings) refresh()
        })
        return () => {
            stopOpenings()
            serve(undefined)
        }
    }, [submitter, signedIn])

    useEffect(() => useFlagSubmissions.subscribe(({submissions}, was) => {
        if (submissions === was.submissions) return
        const told = toldSubmissions()
        if (told === undefined) return
        const fresh = untold(submissions, told)
        if (fresh.length === 0) return
        keepTold(withTold(told, fresh))
        useNews.getState().tellSubmission(fresh[0])
    }), [])
}
