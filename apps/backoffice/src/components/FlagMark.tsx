import {blankFlag, flagOf} from "../countries.ts"

// A flag beside its country's name: its emoji, a plain grey flag for one with none, nothing for Brittany or no id.
export function FlagMark({id}: {id: string}) {
    const emoji = flagOf(id)
    if (emoji !== "") return <span className="flag" aria-hidden="true">{emoji}</span>
    if (blankFlag(id)) return <span className="flag flag-blank" aria-hidden="true"/>
    return null
}
