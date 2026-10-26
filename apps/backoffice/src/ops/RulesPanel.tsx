import {useCallback, useState} from "react"
import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {clockOf, timeOf} from "../components/time.ts"
import {GetRulesResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {wordsOf} from "../refusal.ts"
import {useCall} from "../useCall.ts"
import {
    boxesOf,
    complete,
    Draft,
    draftOf,
    endOverride,
    NEW_DRAFT,
    overrideMinutesValid,
    overrideRules,
    playingAt,
    saveRuleSet,
    tempoOf,
} from "./rules.ts"

type Editing = {draft: Draft, made: boolean}

// What the game plays now and why, an override over it, every rule set postgres keeps, and a form to make or change one.
export function RulesPanel({planet}: {planet: PlanetAdmin}) {
    const read = useCallback((signal: AbortSignal) => planet.getRules({}, {signal}), [planet])
    const [rules, again] = useCall(read)
    const [editing, setEditing] = useState<Editing | undefined>(undefined)

    const saved = () => {
        setEditing(undefined)
        again()
    }

    return (
        <section aria-label="Game rules">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && (
                <>
                    <Playing planet={planet} rules={rules.value} onChanged={again}/>
                    <OverrideForm planet={planet} rules={rules.value} onDone={again}/>
                    <Sets rules={rules.value} onEdit={set => setEditing({draft: draftOf(set), made: true})}/>
                    <Baselines rules={rules.value}/>
                </>
            )}
            <button type="button" className="button" onClick={() => setEditing({draft: NEW_DRAFT, made: false})}>
                New set
            </button>
            {editing !== undefined && (
                <SetForm
                    key={editing.made ? editing.draft.name : "new"}
                    planet={planet}
                    editing={editing}
                    onCancel={() => setEditing(undefined)}
                    onSaved={saved}
                />
            )}
        </section>
    )
}

function Playing({planet, rules, onChanged}: {planet: PlanetAdmin, rules: GetRulesResponse, onChanged: () => void}) {
    const playing = playingAt(rules, new Date())

    return (
        <section className="panel" aria-label="Playing now">
            <h2>Playing <code>{playing.set}</code></h2>
            {playing.layer === "override" && (
                <>
                    <p>An override, until {timeOf(playing.until)}. The baseline plays again then.</p>
                    <ConfirmAction
                        label="End override"
                        question="End the override now? The baseline plays again within a second."
                        run={() => endOverride(planet)}
                        said={() => "Ended."}
                        refusal={wordsOf}
                        onDone={onChanged}
                    />
                </>
            )}
            {playing.layer === "baseline" && (
                <p>
                    The baseline, <code>{playing.request}</code>
                    {playing.since !== undefined && <>, since {timeOf(playing.since)}</>}.
                </p>
            )}
            {playing.layer === "default" && <p>No baseline was ever applied.</p>}
            <p className="hint">
                The game follows a change within a second, for every player. An override plays over the baseline until
                its end, a day at most.
            </p>
        </section>
    )
}

function OverrideForm({planet, rules, onDone}: {planet: PlanetAdmin, rules: GetRulesResponse, onDone: () => void}) {
    const [set, setSet] = useState(rules.sets[0]?.name ?? "")
    const [minutes, setMinutes] = useState("30")
    const valid = set !== "" && overrideMinutesValid(minutes)
    const until = new Date(Date.now() + Number(minutes) * 60_000)

    return (
        <section className="panel" aria-label="Override">
            <h2>Override</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Set</span>
                    <select value={set} onChange={e => setSet(e.target.value)}>
                        {rules.sets.map(ruleSet => <option key={ruleSet.name} value={ruleSet.name}>{ruleSet.name}</option>)}
                    </select>
                </label>
                <label className="field">
                    <span className="muted">For (min)</span>
                    <input
                        type="number"
                        inputMode="numeric"
                        min={1}
                        max={24 * 60}
                        value={minutes}
                        onChange={e => setMinutes(e.target.value)}
                    />
                </label>
            </div>
            <ConfirmAction
                label="Override"
                disabled={!valid}
                question={valid ? `Play ${set} until about ${clockOf(until)}, over the baseline?` : ""}
                run={() => overrideRules(planet, set, Number(minutes), new Date())}
                said={() => "Overridden."}
                refusal={wordsOf}
                onDone={onDone}
            />
        </section>
    )
}

function Baselines({rules}: {rules: GetRulesResponse}) {
    if (rules.baselines.length === 0) return null

    return (
        <table className="table">
            <caption>Baselines applied, the newest first</caption>
            <thead>
                <tr><th>Request</th><th>Set</th><th>Applied</th></tr>
            </thead>
            <tbody>
                {rules.baselines.map(baseline => (
                    <tr key={baseline.requestId}>
                        <td><code>{baseline.requestId}</code></td>
                        <td>{baseline.ruleSet}</td>
                        <td>{baseline.appliedAt !== undefined ? timeOf(baseline.appliedAt.toDate()) : "—"}</td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

function Sets({rules, onEdit}: {rules: GetRulesResponse, onEdit: (set: GetRulesResponse["sets"][number]) => void}) {
    const playing = playingAt(rules, new Date()).set

    return (
        <table className="table">
            <caption>Sets</caption>
            <thead>
                <tr><th>Set</th><th>Refill</th><th>Boxes</th><th/></tr>
            </thead>
            <tbody>
                {rules.sets.map(set => (
                    <tr key={set.name} aria-current={set.name === playing ? "true" : undefined}>
                        <td><strong>{set.name}</strong></td>
                        <td>{tempoOf(set)}</td>
                        <td>{boxesOf(set.boxes)}</td>
                        <td>
                            <button type="button" className="button-mini" onClick={() => onEdit(set)}>
                                Edit {set.name}
                            </button>
                        </td>
                    </tr>
                ))}
            </tbody>
        </table>
    )
}

const FIELDS: {key: Exclude<keyof Draft, "name">, label: string}[] = [
    {key: "tempo", label: "Refill, times as fast"},
    {key: "minSeconds", label: "Shortest box wait (s)"},
    {key: "maxSeconds", label: "Longest box wait (s)"},
    {key: "cap", label: "Charges an hour, at most"},
]

function SetForm({planet, editing, onCancel, onSaved}: {
    planet: PlanetAdmin, editing: Editing, onCancel: () => void, onSaved: () => void,
}) {
    const [draft, setDraft] = useState(editing.draft)
    const change = (part: Partial<Draft>) => setDraft(was => ({...was, ...part}))

    return (
        <section className="panel" aria-label={editing.made ? `Edit ${editing.draft.name}` : "New set"}>
            <h2>{editing.made ? `Edit ${editing.draft.name}` : "New set"}</h2>
            <div className="fields">
                <label className="field">
                    <span className="muted">Name</span>
                    <input value={draft.name} disabled={editing.made} onChange={e => change({name: e.target.value})}/>
                </label>
                {FIELDS.map(({key, label}) => (
                    <label key={key} className="field">
                        <span className="muted">{label}</span>
                        <input
                            type="number"
                            inputMode="decimal"
                            value={draft[key]}
                            onChange={e => change({[key]: e.target.value})}
                        />
                    </label>
                ))}
            </div>
            <ConfirmAction
                label="Save"
                disabled={!complete(draft)}
                question={`Save ${draft.name.trim()}?`}
                run={() => saveRuleSet(planet, draft)}
                said={() => "Saved."}
                refusal={wordsOf}
                onDone={onSaved}
            />
            <button type="button" className="button" onClick={onCancel}>Cancel</button>
        </section>
    )
}
