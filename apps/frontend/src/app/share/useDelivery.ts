import {useEffect, useRef, useState} from "react";

export type DeliveryState<Delivery, Outcome> =
    | {kind: "idle"}
    | {kind: "working", delivery: Delivery}
    | {kind: "done", delivery: Delivery, outcome: Landed<Outcome>}
    | {kind: "failed", delivery: Delivery}

type Landed<Outcome> = Exclude<Outcome, "cancelled">

const SETTLE_MS = 2_500

export function useDelivery<Delivery extends string, Outcome extends string>(
    deliver: (delivery: Delivery) => Promise<Outcome>,
) {
    const [state, setState] = useState<DeliveryState<Delivery, Outcome>>({kind: "idle"})
    const settling = useRef<number | undefined>(undefined)

    useEffect(() => () => window.clearTimeout(settling.current), [])

    const attempt = async (delivery: Delivery): Promise<DeliveryState<Delivery, Outcome>> => {
        try {
            const outcome = await deliver(delivery)
            return landed(outcome) ? {kind: "done", delivery, outcome} : {kind: "idle"}
        } catch (error) {
            console.error(`Failed to deliver the share (${delivery})`, error)
            return {kind: "failed", delivery}
        }
    }

    const run = async (delivery: Delivery) => {
        window.clearTimeout(settling.current)
        setState({kind: "working", delivery})

        const next = await attempt(delivery)
        setState(next)
        if (next.kind === "idle") return

        settling.current = window.setTimeout(() => setState({kind: "idle"}), SETTLE_MS)
    }

    return {state, run}
}

function landed<Outcome extends string>(outcome: Outcome): outcome is Landed<Outcome> {
    return outcome !== "cancelled"
}
