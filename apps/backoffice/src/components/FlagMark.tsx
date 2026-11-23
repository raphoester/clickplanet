import {blankFlag, flagOf, flagPictureOf} from "../countries.ts"

// A flag beside its country's name: its emoji, a sold flag's picture, a plain grey flag for one with neither, nothing
// for Brittany or no id.
export function FlagMark({id}: {id: string}) {
    const emoji = flagOf(id)
    if (emoji !== "") return <span className="flag" aria-hidden="true">{emoji}</span>
    const picture = flagPictureOf(id)
    if (picture !== undefined) return <img className="flag flag-picture" src={picture} alt="" aria-hidden="true"/>
    if (blankFlag(id)) return <span className="flag flag-blank" aria-hidden="true"/>
    return null
}
