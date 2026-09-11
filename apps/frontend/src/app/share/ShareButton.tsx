import {useEffect, useRef, useState} from "react";
import {CapturedFrame} from "../viewer/capture.ts";
import {ShareStats} from "../../domain/shareCard.ts";
import {ShareIcon} from "../components/icons.tsx";
import {ShareOutcome} from "./deliverShare.ts";
import {shareGlobe} from "./shareGlobe.ts";
import "./ShareButton.css"

export type ShareButtonProps = {
    stats: ShareStats
    capture: () => Promise<CapturedFrame>
}

type ShareState =
    | {kind: "idle"}
    | {kind: "working"}
    | {kind: "done", outcome: ShareOutcome}
    | {kind: "failed"}

/** How long the outcome stands before the button offers itself again. */
const SETTLE_MS = 2_500

/**
 * Turns the globe into a picture and gets it out of the browser.
 *
 * The whole outcome is said in the button's own label rather than in a panel
 * beside it: there are four of them, three are good news, and none is worth a
 * line of chrome that sits there empty the rest of the time.
 */
export default function ShareButton({stats, capture}: ShareButtonProps) {
    const [state, setState] = useState<ShareState>({kind: "idle"})
    const settling = useRef<number | undefined>(undefined)

    useEffect(() => () => window.clearTimeout(settling.current), [])

    const share = async () => {
        window.clearTimeout(settling.current)
        setState({kind: "working"})

        const next = await attempt()

        setState(next)
        // A share sheet the player closed themselves has nothing to report, and
        // an idle button is already the whole of what it would have said.
        if (next.kind === "idle") return

        settling.current = window.setTimeout(() => setState({kind: "idle"}), SETTLE_MS)
    }

    const attempt = async (): Promise<ShareState> => {
        try {
            const outcome = await shareGlobe(capture, stats)
            return outcome === "cancelled" ? {kind: "idle"} : {kind: "done", outcome}
        } catch (error) {
            console.error("Failed to share the globe", error)
            return {kind: "failed"}
        }
    }

    return <button type="button"
                   className="button button-ghost button-share"
                   disabled={state.kind === "working"}
                   onClick={share}>
        <ShareIcon/>
        <span aria-live="polite">{label(state)}</span>
    </button>
}

function label(state: ShareState): string {
    switch (state.kind) {
        case "working":
            return "Drawing…"
        case "failed":
            return "Try again"
        case "done":
            return outcomeLabel(state.outcome)
        default:
            return "Share"
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
