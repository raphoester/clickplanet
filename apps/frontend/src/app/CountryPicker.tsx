import {Countries, Country} from "../domain/countries.ts";
import {Allegiance, lockOf} from "../domain/allegiance.ts";
import SelectWithSearch from "./components/SelectWithSearch.tsx";
import "./CountryPicker.css"

export type CountryPickerProps = {
    country: Country,
    setCountry: (country: Country) => void,
    allegiance?: Allegiance,
}

const changeTime = new Intl.DateTimeFormat(undefined, {hour: "2-digit", minute: "2-digit"})

export default function CountryPicker(props: CountryPickerProps) {
    const locked = lockOf(props.allegiance, Date.now())
    if (locked?.changeableAt !== undefined) {
        return <p className="country-picker-locked" role="status">
            You can play for another country at {changeTime.format(locked.changeableAt)}.
        </p>
    }

    return <SelectWithSearch
        onChange={props.setCountry}
        selected={props.country}
        values={Array.from(Countries.values())}
    />
}
