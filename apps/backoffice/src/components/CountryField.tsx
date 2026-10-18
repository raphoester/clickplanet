import {countryIdOf, countryName} from "../countries.ts"

type Props = {
    label: string
    value: string
    onChange: (value: string) => void
    // What an empty field means, for a country the call can do without.
    empty?: string
}

// A country id, with the name it reads as below it, so "dz" is seen to be Algeria before anything runs.
export function CountryField({label, value, onChange, empty}: Props) {
    const typed = countryIdOf(value) !== ""
    const name = countryName(value)

    return (
        <div className="field">
            <label className="field">
                <span className="muted">{label}</span>
                <input
                    className="input-country"
                    value={value}
                    placeholder={empty === undefined ? "fr" : "any"}
                    maxLength={6}
                    autoCapitalize="none"
                    autoComplete="off"
                    spellCheck={false}
                    onChange={event => onChange(event.target.value)}
                />
            </label>
            <span className="hint">
                {typed ? (name ?? "No country this page knows by that code") : (empty ?? "A country code: fr, dz, gb-sct…")}
            </span>
        </div>
    )
}
