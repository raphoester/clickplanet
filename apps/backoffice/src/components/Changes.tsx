// A value a form changes: its name, and what it held and holds.
export type Change = {label: string, before: string, after: string}

export function ChangeList({changes}: {changes: Change[]}) {
    return (
        <ul className="changes" aria-label="What changes">
            {changes.map(change => (
                <li key={change.label}>
                    <span className="change-label">{change.label}</span>{" "}
                    <span className="change-before">{change.before === "" ? "unset" : change.before}</span>
                    {" → "}
                    <strong>{change.after === "" ? "unset" : change.after}</strong>
                </li>
            ))}
        </ul>
    )
}
