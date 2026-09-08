import {Countries, Country} from "../domain/countries.ts";
import SelectWithSearch from "./components/SelectWithSearch.tsx";

export type CountryPickerProps = {
    country: Country,
    setCountry: (country: Country) => void,
}

export default function CountryPicker(props: CountryPickerProps) {
    return <SelectWithSearch
        onChange={props.setCountry}
        selected={props.country}
        values={Array.from(Countries.values())}
    />
}
