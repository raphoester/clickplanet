import {KeyboardEvent, useEffect, useRef, useState} from "react";
import {matchesSearch, type Value, visibleOptions} from "./visibleOptions.ts";
import {SearchIcon} from "./icons.tsx";
import CountryFlag from "./CountryFlag.tsx";
import {nameWithoutFlag} from "../../domain/countries.ts";
import "./SelectWithSearch.css"

type SelectWithSearchProps = {
    selected: Value,
    values: Value[],
    onChange: (value: Value) => void
}

export default function SelectWithSearch(props: SelectWithSearchProps) {
    const [search, setSearch] = useState("")
    const listRef = useRef<HTMLDivElement>(null)

    const options = visibleOptions(props.values, props.selected, search)

    const [activeCode, setActiveCode] = useState(props.selected.code)
    const activeIndex = Math.max(options.findIndex((v) => v.code === activeCode), 0)
    const activeOption: Value | undefined = options[activeIndex]

    useEffect(() => {
        const active = listRef.current?.querySelector("[data-active=true]")
        active?.scrollIntoView?.({block: "nearest"})
    }, [activeCode, options.length])

    const choose = (value: Value) => {
        setSearch("")
        setActiveCode(value.code)
        props.onChange(value)
    }

    const handleSearchChange = (next: string) => {
        setSearch(next)
        const nextOptions = visibleOptions(props.values, props.selected, next)
        const cursor = next.trim() === ""
            ? props.selected
            : nextOptions.find((v) => matchesSearch(v, next)) ?? props.selected
        setActiveCode(cursor.code)
    }

    const handleNavigationKey = (event: KeyboardEvent<HTMLElement>): boolean => {
        if (options.length === 0) return false

        const moveTo = (index: number) => setActiveCode(options[index].code)

        switch (event.key) {
            case "ArrowDown":
                moveTo(Math.min(activeIndex + 1, options.length - 1))
                return true
            case "ArrowUp":
                moveTo(Math.max(activeIndex - 1, 0))
                return true
            case "Home":
                moveTo(0)
                return true
            case "End":
                moveTo(options.length - 1)
                return true
            case "Enter":
                if (activeOption) choose(activeOption)
                return true
        }
        return false
    }

    const handleListKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
        if (event.key === " " && activeOption) {
            event.preventDefault()
            choose(activeOption)
            return
        }
        if (handleNavigationKey(event)) event.preventDefault()
    }

    const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
        if (handleNavigationKey(event)) event.preventDefault()
    }

    return <div className="select-with-search">
        <div className="input-search-field">
            <SearchIcon/>
            <input
                type="text"
                placeholder="Search a country"
                value={search}
                onChange={(event) => handleSearchChange(event.target.value)}
                onKeyDown={handleSearchKeyDown}
                className="input-search"
                autoComplete="off"
            />
        </div>
        <div
            ref={listRef}
            role="listbox"
            tabIndex={0}
            aria-label="Country"
            aria-activedescendant={activeOption && `country-option-${activeOption.code}`}
            onKeyDown={handleListKeyDown}
            className="input-select">
            {options.map((v) => (
                // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/interactive-supports-focus
                <div
                    id={`country-option-${v.code}`}
                    className="input-select-option"
                    key={v.code}
                    role="option"
                    aria-selected={v.code === props.selected.code}
                    data-active={v.code === activeOption?.code}
                    onClick={() => choose(v)}>
                    <CountryFlag code={v.code}/>
                    {nameWithoutFlag(v)}
                </div>
            ))}
        </div>
    </div>
}
