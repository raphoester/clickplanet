import {useEffect, useState} from "react";
import {Countries, Country} from "../domain/countries.ts";
import {Allegiance, lockOf} from "../domain/allegiance.ts";
import {describeEnd} from "../domain/endTime.ts";
import {AllegianceLockedError} from "../backends/backend.ts";
import SelectWithSearch from "./components/SelectWithSearch.tsx";
import "./CountryPicker.css"

export type CountryPickerProps = {
    country: Country,
    setCountry: (country: Country) => Promise<void> | void,
    allegiance?: Allegiance,
}

export const CHANGE_RULE = "You can change country once an hour."

export default function CountryPicker(props: CountryPickerProps) {
    const now = useNowUntilUnlocked(props.allegiance)
    const locked = lockOf(props.allegiance, now)
    const [sending, setSending] = useState(false)
    const [failed, setFailed] = useState(false)

    const pick = async (country: Country) => {
        if (sending) return
        setSending(true)
        setFailed(false)
        try {
            await props.setCountry(country)
        } catch (e) {
            if (!(e instanceof AllegianceLockedError)) {
                console.error("could not change country", e)
                setFailed(true)
            }
        } finally {
            setSending(false)
        }
    }

    return <>
        <p className="country-picker-rule" role="status">
            {CHANGE_RULE}
            {locked?.changeableAt !== undefined && ` Next change at ${describeEnd(locked.changeableAt, now)}.`}
        </p>
        {failed && <p className="country-picker-failed" role="alert">Could not change country. Try again.</p>}
        {!locked && <div className="country-picker-list" aria-busy={sending || undefined}>
            <SelectWithSearch
                onChange={pick}
                selected={props.country}
                values={Array.from(Countries.values())}
            />
        </div>}
    </>
}

function useNowUntilUnlocked(allegiance: Allegiance | undefined): number {
    const [, setUnlocked] = useState(0)

    useEffect(() => {
        const left = (allegiance?.changeableAt ?? 0) - Date.now()
        if (left <= 0) return
        const timer = setTimeout(() => setUnlocked((n) => n + 1), left)
        return () => clearTimeout(timer)
    }, [allegiance])

    return Date.now()
}
