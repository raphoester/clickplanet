import {useState} from "react";
import {CopyIcon, DownloadIcon, ShareIcon} from "../components/icons.tsx";
import {deliveriesOffered, deliverShare, ShareDelivery, ShareOutcome} from "./deliverShare.ts";
import {DeliveryState, useDelivery} from "./useDelivery.ts";
import {t} from "../../i18n/page.ts";
import "./ShareActions.css"

export type ShareActionsProps = {
    file: File
    text: string
    // The ways this browser lets go of the file: a picture's by default.
    offered?: () => ShareDelivery[]
}

type ShareState = DeliveryState<ShareDelivery, ShareOutcome>

export default function ShareActions({file, text, offered = deliveriesOffered}: ShareActionsProps) {
    const [deliveries] = useState(offered)
    const {state, run} = useDelivery((delivery: ShareDelivery) => deliverShare(delivery, file, text))

    return <div className="share-actions">
        {deliveries.map((delivery) =>
            <button key={delivery}
                    type="button"
                    className={`button button-secondary button-share button-share-${delivery}`}
                    disabled={state.kind === "working"}
                    onClick={() => run(delivery)}>
                <Icon delivery={delivery}/>
                <span aria-live="polite">{label(delivery, state)}</span>
            </button>)}
    </div>
}

function Icon({delivery}: {delivery: ShareDelivery}) {
    switch (delivery) {
        case "sheet":
            return <ShareIcon/>
        case "copy":
            return <CopyIcon/>
        case "download":
            return <DownloadIcon/>
    }
}

function label(delivery: ShareDelivery, state: ShareState): string {
    if (state.kind !== "idle" && state.delivery === delivery) {
        switch (state.kind) {
            case "working":
                return t("share.working")
            case "failed":
                return t("common.retry")
            case "done":
                return outcomeLabel(state.outcome)
        }
    }

    return resting(delivery)
}

function resting(delivery: ShareDelivery): string {
    switch (delivery) {
        case "sheet":
            return t("common.share")
        case "copy":
            return t("common.copy")
        case "download":
            return t("share.download")
    }
}

function outcomeLabel(outcome: Exclude<ShareOutcome, "cancelled">): string {
    switch (outcome) {
        case "shared":
            return t("share.shared")
        case "copied":
            return t("common.copied")
        case "downloaded":
            return t("share.downloaded")
    }
}
