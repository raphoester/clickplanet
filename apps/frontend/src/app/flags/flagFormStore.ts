import {create} from "zustand"
import type {Pin} from "../../domain/flags.ts"

// A pin, and the place the page names there: a province or a landmass; none at sea, or for a pin the page was given.
export type PinnedPlace = {pin: Pin, place?: string}

// What the player typed, kept while the form is put aside to aim the pin.
export type Draft = {
    name: string
    file?: File
    pinned?: PinnedPlace
    // The account's submission that waits: the draft takes its place.
    replaces?: string
}

const EMPTY: Draft = {name: ""}

type FlagFormStore = {
    // The form's place is open: its status, or the form; aside while the pin is aimed.
    shown: boolean
    composing: boolean
    aiming: boolean
    draft: Draft
    show: () => void
    compose: (draft?: Draft) => void
    edit: (changes: Partial<Draft>) => void
    aim: () => void
    pinned: (pinned: PinnedPlace) => void
    putAway: () => void
    // Back to the status, which shows what was sent.
    sent: () => void
    close: () => void
}

export const useFlagForm = create<FlagFormStore>()((set) => ({
    shown: false,
    composing: false,
    aiming: false,
    draft: EMPTY,
    show: () => set({shown: true, composing: false, aiming: false, draft: EMPTY}),
    compose: (draft = EMPTY) => set({shown: true, composing: true, aiming: false, draft}),
    edit: (changes) => set(({draft}) => ({draft: {...draft, ...changes}})),
    aim: () => set(({shown}) => shown ? {aiming: true} : {}),
    pinned: (pinned) => set(({draft, aiming}) => aiming ? {aiming: false, draft: {...draft, pinned}} : {}),
    putAway: () => set({aiming: false}),
    sent: () => set({composing: false, aiming: false, draft: EMPTY}),
    close: () => set({shown: false, composing: false, aiming: false, draft: EMPTY}),
}))
