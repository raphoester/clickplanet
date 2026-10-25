import {useCallback} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {When} from "../components/When.tsx"
import {GetRulesResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {applyRuleSet, boxesOf, crowdOf, inForceOf, sourceOf, tempoOf} from "./rules.ts"

// The rule set in force, every set the config holds, and who changed it when.
export function RulesPanel({planet}: {planet: PlanetAdmin}) {
    const read = useCallback((signal: AbortSignal) => planet.getRules({}, {signal}), [planet])
    const [rules, again] = useCall(read)

    return (
        <section aria-label="Game rules">
            <p className="hint">
                Seasons puts the finale&apos;s set in force when it starts, and the default when the season ends. A set
                applied here changes the game for every player at once, until the season&apos;s next phase.
            </p>
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && <Rules rules={rules.value} planet={planet} onChanged={again}/>}
        </section>
    )
}

function Rules({rules, planet, onChanged}: {rules: GetRulesResponse, planet: PlanetAdmin, onChanged: () => void}) {
    const inForce = inForceOf(rules)
    const last = rules.changes[0]

    return (
        <>
            <p role="status">
                In force: <strong>{inForce}</strong>
                {last !== undefined && <> since <When at={last.appliedAt}/>, by {sourceOf(last.source).toLowerCase()}</>}
            </p>
            <button type="button" className="button" onClick={onChanged}>Refresh</button>

            <table className="table">
                <caption>Sets</caption>
                <thead>
                    <tr><th>Set</th><th>Refill</th><th>Boxes</th><th>Anti-bot</th><th/></tr>
                </thead>
                <tbody>
                    {rules.sets.map(set => (
                        <tr key={set.name} aria-current={set.name === inForce ? "true" : undefined}>
                            <td><strong>{set.name}</strong></td>
                            <td>{tempoOf(set)}</td>
                            <td>{boxesOf(set.boxes)}</td>
                            <td>{crowdOf(set)}</td>
                            <td>
                                {set.name === inForce
                                    ? <span className="muted">In force</span>
                                    : <ConfirmAction
                                        label={`Apply ${set.name}`}
                                        question={`Put ${set.name} in force for every player now?`}
                                        run={() => applyRuleSet(planet, set.name)}
                                        said={() => "Applied."}
                                        refusal={wordsOf}
                                        onDone={onChanged}
                                    />}
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>

            <table className="table">
                <caption>Changes, newest first</caption>
                <thead>
                    <tr><th>When</th><th>Set</th><th>By</th><th>Request</th></tr>
                </thead>
                <tbody>
                    {rules.changes.map(change => (
                        <tr key={change.requestId}>
                            <td><When at={change.appliedAt}/></td>
                            <td>{change.ruleSet}</td>
                            <td>{sourceOf(change.source)}</td>
                            <td><code className="mono">{change.requestId}</code></td>
                        </tr>
                    ))}
                    {rules.changes.length === 0 && (
                        <tr><td colSpan={4} className="muted">No change yet: the default has always been in force.</td></tr>
                    )}
                </tbody>
            </table>
        </>
    )
}
