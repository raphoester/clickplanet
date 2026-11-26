import {create} from "zustand"
import type {Pin} from "../../domain/flags.ts"
import {flagRowsOf, replacedByNewOf, Submission} from "../../domain/flagSubmissions.ts"
import {useDismissedFlagRows} from "./dismissedFlagRows.ts"
import {useFlags} from "./flagsStore.ts"
import {useFlagSubmissions} from "./submissionsStore.ts"

// A pin, and the place the page names there: a province or a landmass; none at sea, or for a pin the page was given.
export type PinnedPlace = {pin: Pin, place?: string}

// What the player typed, kept while the form is put aside to aim the pin, and on the paywall for Back and Fix it.
export type Draft = {
    name: string
    file?: File
    pinned?: PinnedPlace
    // The account's submission that waits: the draft takes its place.
    replaces?: string
    // For a new flag, not a change: the name of the one that waits unpaid, which it replaces.
    replacing?: string
}

// The shop's door opens the account's flags once they are read, or a new one when it has none; the picker's doors a
// new flag. A flag is made, then paid; a flag held gets its time, and one whose time ended is claimed, on its own page.
export type Step =
    | {kind: "door", to: "hub" | "make"}
    | {kind: "hub"}
    | {kind: "make"}
    // As sent, and what was typed for it: a read may come back before the server lists it, and the page keeps its
    // picture alone.
    | {kind: "pay", submission: Submission, typed?: Draft}
    // Add time to a flag held, or claim one whose time ended. fromHub: Back goes to the account's flags.
    | {kind: "flag", flag: string, fromHub: boolean}

const EMPTY: Draft = {name: ""}

type FlagFormStore = {
    shown: boolean
    step: Step
    aiming: boolean
    draft: Draft
    // A new flag: the picker's row, and the picker's search that found no country, as typed.
    open: (typed?: string) => void
    // The shop's door: the account's flags, or a new one when it has none.
    openShop: () => void
    // From the account's flags: a new flag, which takes the place of the one that waits unpaid.
    addNew: () => void
    // The pay page of a flag: its holder's Add time, or a claim.
    payFor: (flag: string, fromHub?: boolean) => void
    // Back to the account's flags.
    toHub: () => void
    // The door's step, once the submissions are read.
    land: () => void
    compose: (draft?: Draft) => void
    edit: (changes: Partial<Draft>) => void
    aim: () => void
    pinned: (pinned: PinnedPlace) => void
    putAway: () => void
    // Step 2 for a flag: the one Next sent, reviewed while it is paid for, or the one a change found reviewed already.
    showPaywall: (submission: Submission, typed?: Draft) => void
    // Back to the flag sent: a change of the one that waits, or after a rejection a new one, with what was typed.
    back: (submission: Submission) => void
    close: () => void
}

function landed(to: "hub" | "make", typed: string): Pick<FlagFormStore, "step" | "draft"> {
    const {read, submissions, held} = useFlagSubmissions.getState()
    if (!read) return {step: {kind: "door", to}, draft: {name: typed}}
    const rows = flagRowsOf({submissions, held}, useFlags.getState().catalog, useDismissedFlagRows.getState().dismissed)
    if (to === "hub" && rows.length > 0) return {step: {kind: "hub"}, draft: EMPTY}
    return {step: {kind: "make"}, draft: newDraftOf(typed, submissions)}
}

// One submission waits at a time: a new flag takes the place of the one that waits unpaid, and says so.
function newDraftOf(name: string, submissions: readonly Submission[]): Draft {
    const replaced = replacedByNewOf(submissions)
    return replaced ? {name, replaces: replaced.id, replacing: replaced.name} : {name}
}

// The picture is picked again once the page forgot it: it keeps none.
function againOf(submission: Submission, typed: Draft): Draft {
    if (typed.file) return {...typed, replaces: undefined, replacing: undefined}
    return {name: submission.name, pinned: submission.pin && {pin: submission.pin}}
}

export const useFlagForm = create<FlagFormStore>()((set) => ({
    shown: false,
    step: {kind: "door", to: "make"},
    aiming: false,
    draft: EMPTY,
    open: (typed = "") => set({shown: true, aiming: false, ...landed("make", typed.trim())}),
    openShop: () => set({shown: true, aiming: false, ...landed("hub", "")}),
    addNew: () => set({
        shown: true, aiming: false, step: {kind: "make"},
        draft: newDraftOf("", useFlagSubmissions.getState().submissions),
    }),
    payFor: (flag, fromHub = false) => set({shown: true, aiming: false, step: {kind: "flag", flag, fromHub}, draft: EMPTY}),
    toHub: () => set({shown: true, aiming: false, step: {kind: "hub"}, draft: EMPTY}),
    land: () => set(({step, draft}) => step.kind === "door" ? landed(step.to, draft.name) : {}),
    compose: (draft = EMPTY) => set({shown: true, step: {kind: "make"}, aiming: false, draft}),
    edit: (changes) => set(({draft}) => ({draft: {...draft, ...changes}})),
    aim: () => set(({shown}) => shown ? {aiming: true} : {}),
    pinned: (pinned) => set(({draft, aiming}) => aiming ? {aiming: false, draft: {...draft, pinned}} : {}),
    putAway: () => set({aiming: false}),
    showPaywall: (submission, typed) => set(({draft}) => ({
        shown: true,
        step: {kind: "pay", submission, typed},
        aiming: false,
        draft: {...draft, replaces: undefined, replacing: undefined},
    })),
    back: (submission) => set(({draft}) => {
        const again = againOf(submission, draft)
        return {step: {kind: "make"}, draft: submission.state === "waiting" ? {...again, replaces: submission.id} : again}
    }),
    close: () => set({shown: false, step: {kind: "door", to: "make"}, aiming: false, draft: EMPTY}),
}))
