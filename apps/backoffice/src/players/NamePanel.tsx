import {useState} from "react"
import {PlayerAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {refusalOf, rename} from "./actions.ts"

type Props = {player: PlayerAdmin, accountId: string, who: string, onDone: () => void}

export function NamePanel({player, accountId, who, onDone}: Props) {
    const [typed, setTyped] = useState("")
    const name = typed.trim()

    return (
        <div className="tool-body">
            <label className="field">
                <span className="muted">New name</span>
                <input type="text" value={typed} onChange={e => setTyped(e.target.value)}/>
            </label>
            <ConfirmAction
                label="Rename"
                disabled={name === ""}
                question={`Rename ${who} to ${name}?`}
                run={() => rename(player, accountId, name)}
                said={kept => `Renamed to ${kept}.`}
                refusal={err => refusalOf("rename", err)}
                onDone={onDone}
            />
        </div>
    )
}
