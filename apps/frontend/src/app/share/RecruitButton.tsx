import {ReactNode, useState} from "react";
import {shareUrl} from "../../domain/shareCard.ts";
import {CopyIcon, ShareIcon} from "../components/icons.tsx";
import {deliverLink, LinkDelivery, linkDeliveryOffered, LinkOutcome} from "./deliverShare.ts";
import {DeliveryState, useDelivery} from "./useDelivery.ts";

export type RecruitButtonProps = {
    code: string
    text: string
    className: string
    icon?: ReactNode
    label?: string
}

type LinkState = DeliveryState<LinkDelivery, LinkOutcome>

export default function RecruitButton({code, text, className, icon, label}: RecruitButtonProps) {
    const [delivery] = useState(linkDeliveryOffered)
    const {state, run} = useDelivery((offered: LinkDelivery) => deliverLink(offered, shareUrl(code), text))

    return <button type="button"
                   className={className}
                   disabled={state.kind === "working"}
                   onClick={() => run(delivery)}>
        {icon ?? <Icon delivery={delivery}/>}
        <span aria-live="polite">{state.kind === "idle" && label ? label : stateLabel(delivery, state)}</span>
    </button>
}

function Icon({delivery}: {delivery: LinkDelivery}) {
    switch (delivery) {
        case "sheet":
            return <ShareIcon/>
        case "copy":
            return <CopyIcon/>
    }
}

function stateLabel(delivery: LinkDelivery, state: LinkState): string {
    switch (state.kind) {
        case "idle":
            return resting(delivery)
        case "working":
            return "One sec…"
        case "failed":
            return "Try again"
        case "done":
            return outcomeLabel(state.outcome)
    }
}

function resting(delivery: LinkDelivery): string {
    switch (delivery) {
        case "sheet":
            return "Share link"
        case "copy":
            return "Copy link"
    }
}

function outcomeLabel(outcome: Exclude<LinkOutcome, "cancelled">): string {
    switch (outcome) {
        case "shared":
            return "Shared!"
        case "copied":
            return "Copied!"
    }
}
