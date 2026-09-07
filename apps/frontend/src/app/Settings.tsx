import {Countries, Country} from "../domain/countries.ts";
import SelectWithSearch from "./components/SelectWithSearch.tsx";
import ModalManager from "./components/ModalManager.tsx";

type SettingsProps = {
    country: Country,
    setCountry: (country: Country) => void
}

export default function Settings(props: SettingsProps) {
    return <ModalManager
        modalTitle="Country"
        buttonProps={{
            className: "button-settings",
            text: props.country.name,
            imageUrl: `/static/countries/svg/${props.country.code}.svg`,
        }}
    >
        <SelectWithSearch
            onChange={props.setCountry}
            selected={props.country}
            values={Array.from(Countries.values())}
        />
    </ModalManager>
}
