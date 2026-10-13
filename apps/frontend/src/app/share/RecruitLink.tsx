import {useState} from "react";
import {shareHeadline, shareLabel, ShareStats, shareUrl} from "../../domain/shareCard.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import {CopyIcon, ShareIcon} from "../components/icons.tsx";
import {deliverLink, LinkDelivery, linkDeliveryOffered, LinkOutcome} from "./deliverShare.ts";
import {DeliveryState, useDelivery} from "./useDelivery.ts";
import "./RecruitLink.css"

export type RecruitLinkProps = {
    stats: ShareStats
}

type LinkState = DeliveryState<LinkDelivery, LinkOutcome>

export default function RecruitLink({stats}: RecruitLinkProps) {
    const [delivery] = useState(linkDeliveryOffered)
    const code = stats.country.code
    const {state, run} = useDelivery((offered: LinkDelivery) =>
        deliverLink(offered, shareUrl(code), shareHeadline(stats)))

    return <div className="panel-box recruit">
        <span className="recruit-title">
            <CountryFlag code={code}/>
            <span className="recruit-country">Recruit for {stats.country.name}</span>
        </span>
        <span className="recruit-link">{shareLabel(code)}</span>
        <button type="button"
                className="button button-secondary recruit-send"
                disabled={state.kind === "working"}
                onClick={() => run(delivery)}>
            <Icon delivery={delivery}/>
            <span aria-live="polite">{label(delivery, state)}</span>
        </button>
    </div>
}

function Icon({delivery}: {delivery: LinkDelivery}) {
    switch (delivery) {
        case "sheet":
            return <ShareIcon/>
        case "copy":
            return <CopyIcon/>
    }
}

function label(delivery: LinkDelivery, state: LinkState): string {
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
