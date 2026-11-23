import {countryLabel, flagged} from "../countries.ts"
import {FlagMark} from "./FlagMark.tsx"

// A country as the operator reads it: its flag, where the page has one, and its name.
export function Country({id}: {id: string}) {
    return (
        <span className="country">
            {flagged(id) && <><FlagMark id={id}/> </>}
            {countryLabel(id)}
        </span>
    )
}
