import {useEffect, useRef, useState} from "react";
import {CapturedFrame} from "../viewer/capture.ts";
import {ShareStats} from "../../domain/shareCard.ts";
import {CopyIcon, DownloadIcon, ShareIcon} from "../components/icons.tsx";
import {deliveriesOffered, ShareDelivery, ShareOutcome} from "./deliverShare.ts";
import {shareGlobe} from "./shareGlobe.ts";
import "./ShareActions.css"

export type ShareActionsProps = {
    stats: ShareStats
    capture: () => Promise<CapturedFrame>
}

type ShareState =
    | {kind: "idle"}
    | {kind: "working", delivery: ShareDelivery}
    | {kind: "done", delivery: ShareDelivery, outcome: ShareOutcome}
    | {kind: "failed", delivery: ShareDelivery}

/** How long an outcome stands before the button offers itself again. */
const SETTLE_MS = 2_500

/**
 * Turning the globe into a picture, and the ways this browser has of letting go
 * of it. A phone gets one button and its share sheet; a desktop gets the two
 * that cannot lie about what they did. See `deliveriesOffered`.
 *
 * Each button says its own outcome in its own label rather than in a line of
 * chrome beside them, which would sit there empty the rest of the time. Only
 * the button that was pressed says anything — which is also what tells the
 * player which of the two they actually got.
 */
export default function ShareActions({stats, capture}: ShareActionsProps) {
    // Nothing about what this browser can do changes while the page is open.
    const [deliveries] = useState(deliveriesOffered)
    const [state, setState] = useState<ShareState>({kind: "idle"})
    const settling = useRef<number | undefined>(undefined)

    useEffect(() => () => window.clearTimeout(settling.current), [])

    const run = async (delivery: ShareDelivery) => {
        window.clearTimeout(settling.current)
        setState({kind: "working", delivery})

        const next = await attempt(delivery)
        setState(next)
        // A share sheet the player closed has nothing to report, and an idle
        // button is already the whole of what it would have said.
        if (next.kind === "idle") return

        settling.current = window.setTimeout(() => setState({kind: "idle"}), SETTLE_MS)
    }

    const attempt = async (delivery: ShareDelivery): Promise<ShareState> => {
        try {
            const outcome = await shareGlobe(capture, stats, delivery)
            return outcome === "cancelled" ? {kind: "idle"} : {kind: "done", delivery, outcome}
        } catch (error) {
            console.error(`Failed to share the globe (${delivery})`, error)
            return {kind: "failed", delivery}
        }
    }

    return <div className="share-actions">
        {deliveries.map((delivery) =>
            <button key={delivery}
                    type="button"
                    className={`button button-ghost button-share button-share-${delivery}`}
                    // Both, not just the one pressed: there is one globe to
                    // capture and one frame it is captured from.
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
                return "One sec…"
            case "failed":
                return "Try again"
            case "done":
                return outcomeLabel(state.outcome)
        }
    }

    return resting(delivery)
}

function resting(delivery: ShareDelivery): string {
    switch (delivery) {
        case "sheet":
            return "Share"
        case "copy":
            return "Copy"
        case "download":
            return "Save"
    }
}

function outcomeLabel(outcome: ShareOutcome): string {
    switch (outcome) {
        case "shared":
            return "Shared!"
        case "copied":
            return "Copied!"
        case "downloaded":
            return "Saved!"
        default:
            return "Share"
    }
}
