import {blankFlag, countryLabel, flagOf} from "../countries.ts"
import {FlagMark} from "./FlagMark.tsx"

// A country as the operator reads it: its flag, where Unicode has one, and its name.
export function Country({id}: {id: string}) {
    const flagged = flagOf(id) !== "" || blankFlag(id)
    return (
        <span className="country">
            {flagged && <><FlagMark id={id}/> </>}
            {countryLabel(id)}
        </span>
    )
}
