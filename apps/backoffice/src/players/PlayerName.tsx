import {countryLabel} from "../countries.ts"
import {FlagMark} from "../components/FlagMark.tsx"

type Props = {
    name: string
    // The flag the player paints most, or "".
    flag: string
    // What stands in for a name the page does not have.
    unnamed: string
}

// Who a player is, as a person reads it: its flag and its name, never an id.
export function PlayerName({name, flag, unnamed}: Props) {
    return (
        <span className="player-name">
            <FlagMark id={flag}/>
            <span className={name === "" ? "muted" : "name"}>{name === "" ? unnamed : name}</span>
            {flag !== "" && <> <span className="muted">{countryLabel(flag)}</span></>}
        </span>
    )
}
