import {create} from "zustand"
import type {Pin} from "../../domain/flags.ts"
import {onItsWayOf, Submission} from "../../domain/flagSubmissions.ts"
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
}

// A door opens the flag on its way once the submissions are read, else a new one; then the flag is made, then paid.
export type Step =
    | {kind: "door"}
    | {kind: "make"}
    // As sent, and what was typed for it: a read may come back before the server lists it, and the page keeps its
    // picture alone.
    | {kind: "pay", submission: Submission, typed?: Draft}

const EMPTY: Draft = {name: ""}

type FlagFormStore = {
    shown: boolean
    step: Step
    aiming: boolean
    draft: Draft
    // Every door: the shop's, the picker's row, and the picker's search that found no country, as typed.
    open: (typed?: string) => void
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

function landed(typed: string): Pick<FlagFormStore, "step" | "draft"> {
    const {read, submissions} = useFlagSubmissions.getState()
    if (!read) return {step: {kind: "door"}, draft: {name: typed}}
    const onItsWay = onItsWayOf(submissions)
    return onItsWay ? {step: {kind: "pay", submission: onItsWay}, draft: EMPTY} : {step: {kind: "make"}, draft: {name: typed}}
}

// The picture is picked again once the page forgot it: it keeps none.
function againOf(submission: Submission, typed: Draft): Draft {
    if (typed.file) return {...typed, replaces: undefined}
    return {name: submission.name, pinned: submission.pin && {pin: submission.pin}}
}

export const useFlagForm = create<FlagFormStore>()((set) => ({
    shown: false,
    step: {kind: "door"},
    aiming: false,
    draft: EMPTY,
    open: (typed = "") => set({shown: true, aiming: false, ...landed(typed.trim())}),
    land: () => set(({step, draft}) => step.kind === "door" ? landed(draft.name) : {}),
    compose: (draft = EMPTY) => set({shown: true, step: {kind: "make"}, aiming: false, draft}),
    edit: (changes) => set(({draft}) => ({draft: {...draft, ...changes}})),
    aim: () => set(({shown}) => shown ? {aiming: true} : {}),
    pinned: (pinned) => set(({draft, aiming}) => aiming ? {aiming: false, draft: {...draft, pinned}} : {}),
    putAway: () => set({aiming: false}),
    showPaywall: (submission, typed) => set(({draft}) => ({
        shown: true,
        step: {kind: "pay", submission, typed},
        aiming: false,
        draft: {...draft, replaces: undefined},
    })),
    back: (submission) => set(({draft}) => {
        const again = againOf(submission, draft)
        return {step: {kind: "make"}, draft: submission.state === "waiting" ? {...again, replaces: submission.id} : again}
    }),
    close: () => set({shown: false, step: {kind: "door"}, aiming: false, draft: EMPTY}),
}))
