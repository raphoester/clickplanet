import {useCallback, useState} from "react"
import {MissionsAdmin} from "../clients.ts"
import {ChangeList} from "../components/Changes.tsx"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {FailureNotice} from "../components/FailureNotice.tsx"
import {wordsOf} from "../refusal.ts"
import {changesWord} from "../support/labels.ts"
import {useCall} from "../useCall.ts"
import {changesOf, Draft, draftOf, FIELDS, flawOf, rulesOf, saveRules} from "./missionRules.ts"

// The numbers missions draws with: one document, saved whole.
export function MissionRulesPanel({missions}: {missions: MissionsAdmin}) {
    const read = useCallback((signal: AbortSignal) => missions.getRules({}, {signal}), [missions])
    const [rules, again] = useCall(read)

    return (
        <section aria-label="Mission rules" className="detail">
            {rules.state === "loading" && <p className="muted">Loading…</p>}
            {rules.state === "failed" && <FailureNotice failure={rules} onRetry={again}/>}
            {rules.state === "done" && <FlagMissionForm missions={missions} read={draftOf(rules.value.rules)}/>}
        </section>
    )
}

// What a save kept is what the next save changes from, so the form stays, with what the server said.
function FlagMissionForm({missions, read}: {missions: MissionsAdmin, read: Draft}) {
    const [kept, setKept] = useState(read)
    const [draft, setDraft] = useState(read)
    const changes = changesOf(kept, draft)
    const flaw = flawOf(draft)

    return (
        <>
            <fieldset className="rule-group">
                <legend>Flag missions</legend>
                <div className="fields">
                    {FIELDS.map(field => (
                        <label key={field.key} className="field">
                            <span className="muted">{field.label}</span>
                            <input type="number" inputMode="numeric" min={0} value={draft[field.key]}
                                   onChange={e => setDraft(was => ({...was, [field.key]: e.target.value}))}/>
                        </label>
                    ))}
                </div>
            </fieldset>
            {flaw !== undefined && <p className="action-refused" role="alert">{flaw}</p>}
            <div className="save-bar">
                <span className={changes.length === 0 ? "muted" : "save-count"}>{changesWord(changes.length)}</span>
                <ConfirmAction
                    label="Save"
                    disabled={flaw !== undefined || changes.length === 0}
                    question={(
                        <>
                            <p>Draw every flag mission with these numbers? A flag draws its mission once a day, so most
                                flags see them from the next day.</p>
                            <ChangeList changes={changes}/>
                        </>
                    )}
                    run={() => saveRules(missions, draft)}
                    said={() => "Saved."}
                    refusal={wordsOf}
                    onDone={() => setKept(draftOf(rulesOf(draft)))}
                />
            </div>
        </>
    )
}
