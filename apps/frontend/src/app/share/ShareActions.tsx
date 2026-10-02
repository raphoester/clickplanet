import {useEffect, useRef, useState} from "react";
import {CopyIcon, DownloadIcon, ShareIcon} from "../components/icons.tsx";
import {deliveriesOffered, deliverShare, ShareDelivery, ShareOutcome} from "./deliverShare.ts";
import "./ShareActions.css"

export type ShareActionsProps = {
    file: File
    text: string
}

type ShareState =
    | {kind: "idle"}
    | {kind: "working", delivery: ShareDelivery}
    | {kind: "done", delivery: ShareDelivery, outcome: ShareOutcome}
    | {kind: "failed", delivery: ShareDelivery}

const SETTLE_MS = 2_500

export default function ShareActions({file, text}: ShareActionsProps) {
    const [deliveries] = useState(deliveriesOffered)
    const [state, setState] = useState<ShareState>({kind: "idle"})
    const settling = useRef<number | undefined>(undefined)

    useEffect(() => () => window.clearTimeout(settling.current), [])

    const run = async (delivery: ShareDelivery) => {
        window.clearTimeout(settling.current)
        setState({kind: "working", delivery})

        const next = await attempt(delivery)
        setState(next)
        if (next.kind === "idle") return

        settling.current = window.setTimeout(() => setState({kind: "idle"}), SETTLE_MS)
    }

    const attempt = async (delivery: ShareDelivery): Promise<ShareState> => {
        try {
            const outcome = await deliverShare(delivery, file, text)
            return outcome === "cancelled" ? {kind: "idle"} : {kind: "done", delivery, outcome}
        } catch (error) {
            console.error(`Failed to deliver the picture (${delivery})`, error)
            return {kind: "failed", delivery}
        }
    }

    return <div className="share-actions">
        {deliveries.map((delivery) =>
            <button key={delivery}
                    type="button"
                    className={`button button-ghost button-share button-share-${delivery}`}
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
