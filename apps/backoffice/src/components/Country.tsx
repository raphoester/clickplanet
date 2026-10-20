import {countryLabel, flagOf} from "../countries.ts"

// A country as the operator reads it: its flag, where Unicode has one, and its name.
export function Country({id}: {id: string}) {
    const flag = flagOf(id)
    return (
        <span className="country">
            {flag !== "" && <><span className="flag" aria-hidden="true">{flag}</span> </>}
            {countryLabel(id)}
        </span>
    )
}
