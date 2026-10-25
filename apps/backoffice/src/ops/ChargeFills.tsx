import {useCallback, useEffect} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {GetChargeFillsResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall, useLast} from "../useCall.ts"
import {fillCharges, fillSourceOf, runningOf, stateOf} from "./fills.ts"

export const FOLLOW_MS = 2_000

// Every fill of every player's charges, newest first, read again while one runs.
export function ChargeFills({planet}: {planet: PlanetAdmin}) {
    const read = useCallback((signal: AbortSignal) => planet.getChargeFills({}, {signal}), [planet])
    const [call, again] = useCall(read)
    const fills = useLast(call)
    const running = fills !== undefined && runningOf(fills)

    useEffect(() => {
        if (!running) return
        const timer = setInterval(again, FOLLOW_MS)
        return () => clearInterval(timer)
    }, [running, again])

    return (
        <section aria-label="Charge fills">
            {fills === undefined && call.state === "loading" && <p className="muted">Loading…</p>}
            {call.state === "failed" && <FailureNotice failure={call} onRetry={again}/>}
            {fills !== undefined && <Fills fills={fills} planet={planet} onStarted={again}/>}
        </section>
    )
}

function Fills({fills, planet, onStarted}: {fills: GetChargeFillsResponse, planet: PlanetAdmin, onStarted: () => void}) {
    const latest = fills.fills[0]

    return (
        <>
            <p role="status">
                {latest === undefined
                    ? "No fill yet."
                    : <>Last fill: <strong>{stateOf(latest)}</strong>, {latest.accountsFilled} accounts filled</>}
            </p>
            <ConfirmAction
                label="Fill every bank"
                question="Fill every player's charges to full now?"
                run={() => fillCharges(planet)}
                said={() => "Started."}
                refusal={wordsOf}
                onDone={onStarted}
            />

            {latest !== undefined && (
                <table className="table">
                    <caption>Fills, newest first</caption>
                    <thead>
                        <tr><th>Started</th><th>State</th><th>Accounts filled</th><th>By</th><th>Request</th></tr>
                    </thead>
                    <tbody>
                        {fills.fills.map(fill => (
                            <tr key={fill.requestId}>
                                <td><When at={fill.startedAt}/></td>
                                <td>{stateOf(fill)}</td>
                                <td>{fill.accountsFilled}</td>
                                <td>{fillSourceOf(fill.source)}</td>
                                <td><code className="mono">{fill.requestId}</code></td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </>
    )
}
