import {create} from "zustand"
import {Allegiance} from "../../domain/allegiance.ts"
import {BUBBLE_MS} from "../components/Bubble.tsx"

type RefusedClicksStore = {
    // One more for each click the page turned down: the meter shakes.
    refusals: number
    // One more for each click the bank or a frozen map turned down.
    dryTaps: number
    vpnBlocked: boolean
    bannedUntil?: number
    sessionUnavailable: boolean
    lockedOut?: {allegiance: Allegiance, id: number}
    outOfReach?: {id: number}
    refuse: () => void
    blockVPN: () => void
    ban: (until: number) => void
    failSession: () => void
    lockOut: (allegiance: Allegiance) => void
    refuseOutOfReach: () => void
    dismissVPNBlocked: () => void
    dismissBanned: () => void
    dismissSessionUnavailable: () => void
}

export const useRefusedClicks = create<RefusedClicksStore>()((set) => {
    let lockOuts = 0
    let farOffs = 0

    return {
        refusals: 0,
        dryTaps: 0,
        vpnBlocked: false,
        sessionUnavailable: false,
        refuse: () => set(({refusals, dryTaps}) => ({refusals: refusals + 1, dryTaps: dryTaps + 1})),
        blockVPN: () => set({vpnBlocked: true}),
        ban: (until) => set({bannedUntil: until}),
        failSession: () => set({sessionUnavailable: true}),
        lockOut: (allegiance) => {
            const lockedOut = {allegiance, id: ++lockOuts}
            set({lockedOut})
            setTimeout(() => set((refused) => refused.lockedOut === lockedOut ? {lockedOut: undefined} : refused), BUBBLE_MS)
        },
        refuseOutOfReach: () => {
            const outOfReach = {id: ++farOffs}
            set(({refusals}) => ({outOfReach, refusals: refusals + 1}))
            setTimeout(() => set((refused) => refused.outOfReach === outOfReach ? {outOfReach: undefined} : refused), BUBBLE_MS)
        },
        dismissVPNBlocked: () => set({vpnBlocked: false}),
        dismissBanned: () => set({bannedUntil: undefined}),
        dismissSessionUnavailable: () => set({sessionUnavailable: false}),
    }
})
