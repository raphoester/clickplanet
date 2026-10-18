import {countryLabel, flagOf} from "../countries.ts"

type Props = {
    name: string
    // The flag the player paints most, or "".
    flag: string
    // What stands in for a name the page does not have.
    unnamed: string
}

// Who a player is, as a person reads it: its flag and its name, never an id.
export function PlayerName({name, flag, unnamed}: Props) {
    const emoji = flagOf(flag)
    return (
        <span className="player-name">
            {emoji !== "" && <span className="flag" aria-hidden="true">{emoji}</span>}
            <span className={name === "" ? "muted" : "name"}>{name === "" ? unnamed : name}</span>
            {flag !== "" && <> <span className="muted">{countryLabel(flag)}</span></>}
        </span>
    )
}
