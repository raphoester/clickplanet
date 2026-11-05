import {ReactNode} from "react";
import {InviteLink} from "../../backends/crazyGames.ts";
import {CopyIcon} from "../components/icons.tsx";
import {copyLink, LinkOutcome} from "./deliverShare.ts";
import {DeliveryState, useDelivery} from "./useDelivery.ts";

export type InviteButtonProps = {
    flag: string
    invite: InviteLink
    className: string
    label: string
    icon?: ReactNode
}

type InviteState = DeliveryState<"copy", LinkOutcome>

export default function InviteButton({flag, invite, className, label, icon}: InviteButtonProps) {
    const {state, run} = useDelivery<"copy", LinkOutcome>(async () => {
        const link = await invite(flag)
        if (!link) throw new Error("CrazyGames gave no invite link")
        return copyLink(link)
    })

    return <button type="button"
                   className={className}
                   disabled={state.kind === "working"}
                   onClick={() => run("copy")}>
        {icon ?? <CopyIcon/>}
        <span aria-live="polite">{stateLabel(state, label)}</span>
    </button>
}

function stateLabel(state: InviteState, label: string): string {
    switch (state.kind) {
        case "idle":
            return label
        case "working":
            return "One sec…"
        case "failed":
            return "Try again"
        case "done":
            return "Copied!"
    }
}
