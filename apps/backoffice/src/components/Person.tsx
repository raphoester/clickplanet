import {AccountId} from "./AccountId.tsx"

export function Person({label, name, id}: {label: string, name: string, id: string}) {
    return (
        <div>
            <dt>{label}</dt>
            <dd>
                <span className={name === "" ? "muted" : "name"}>{name === "" ? "no name" : name}</span>
                <AccountId id={id}/>
            </dd>
        </div>
    )
}
