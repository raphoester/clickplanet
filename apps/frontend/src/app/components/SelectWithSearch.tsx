import {ChangeEvent, useState} from "react";
import "./SelectWithSearch.css"

type Value = {
    code: string,
    name: string
}

type SelectWithSearchProps = {
    selected: Value,
    values: Value[],
    onChange: (value: Value) => void
}

/**
 * The selection itself is the caller's state, not ours: it is persisted and
 * pushed to the renderer. Mirroring it locally, as this used to, meant the
 * dropdown kept showing its own stale copy whenever the country changed from
 * anywhere else.
 */
export default function SelectWithSearch(props: SelectWithSearchProps) {
    const [search, setSearch] = useState("")

    const filteredOptions = props.values.filter(
        (v) => v.name.toLowerCase().includes(search.toLowerCase())
    )

    const handleSelectChange = (event: ChangeEvent<HTMLSelectElement>) => {
        const value = props.values.find((v) => v.code === event.target.value)
        if (!value) {
            console.error(`Value not found for code ${event.target.value}`)
            return
        }

        setSearch("")
        props.onChange(value)
    }

    return <div>
        <select
            value={props.selected.code}
            onChange={handleSelectChange}
            size={5}
            className="input-select">
            {filteredOptions.map((v) => (
                <option className="input-select-option" key={v.code} value={v.code}>
                    {v.name}
                </option>
            ))}
        </select>
        <input
            type="text"
            placeholder={"🔍 Search..."}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="input-search"
            autoComplete="off"
        />
    </div>
}
