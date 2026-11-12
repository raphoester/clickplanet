import {ReactNode, useState} from "react";
import {ShareAddress, shareUrl} from "../../domain/shareCard.ts";
import {CopyIcon, ShareIcon} from "../components/icons.tsx";
import {deliverLink, LinkDelivery, linkDeliveryOffered, LinkOutcome} from "./deliverShare.ts";
import {DeliveryState, useDelivery} from "./useDelivery.ts";
import {t} from "../../i18n/page.ts";

export type RecruitButtonProps = {
    address: ShareAddress
    text: string
    className: string
    icon?: ReactNode
    label?: string
}

type LinkState = DeliveryState<LinkDelivery, LinkOutcome>

export default function RecruitButton({address, text, className, icon, label}: RecruitButtonProps) {
    const [delivery] = useState(linkDeliveryOffered)
    const {state, run} = useDelivery((offered: LinkDelivery) => deliverLink(offered, shareUrl(address), text))

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
            return t("share.working")
        case "failed":
            return t("common.retry")
        case "done":
            return outcomeLabel(state.outcome)
    }
}

function resting(delivery: LinkDelivery): string {
    switch (delivery) {
        case "sheet":
            return t("share.shareLink")
        case "copy":
            return t("share.copyLink")
    }
}

function outcomeLabel(outcome: Exclude<LinkOutcome, "cancelled">): string {
    switch (outcome) {
        case "shared":
            return t("share.shared")
        case "copied":
            return t("common.copied")
    }
}
