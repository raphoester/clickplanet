import {PlanetAdmin} from "../clients.ts"
import {ConfirmAction} from "../components/ConfirmAction.tsx"
import {refusalOf, release, Released} from "./actions.ts"

type Props = {planet: PlanetAdmin, accountId: string, who: string}

function releasedWords(released: Released): string {
    return released.locked
        ? "Done, but the rules in force lock every flag: the change waits for them to end."
        : "Done. The next change of flag goes at once."
}

export function FlagPanel({planet, accountId, who}: Props) {
    return (
        <div className="act">
            <ConfirmAction
                label="Allow a change now"
                question={`Let ${who} change flag now, without the hour of wait?`}
                run={() => release(planet, accountId)}
                said={releasedWords}
                refusal={err => refusalOf("release", err)}
            />
        </div>
    )
}
