import {KeyboardEvent, useEffect, useRef, useState} from "react";
import {matchesSearch, type Value, visibleOptions} from "./visibleOptions.ts";
import "./SelectWithSearch.css"

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
 *
 * The list is our own markup rather than a `<select size={5}>`, because mobile
 * WebKit ignores `size` entirely: every `<select>` collapses to a one-line
 * control that opens the native picker, so the search field ended up filtering
 * a list the user could not see, and the collapsed row rendered in the UA's
 * default black on our near-black modal.
 */
export default function SelectWithSearch(props: SelectWithSearchProps) {
    const [search, setSearch] = useState("")
    const listRef = useRef<HTMLDivElement>(null)

    const options = visibleOptions(props.values, props.selected, search)

    // The keyboard cursor, held by code rather than by index so that filtering
    // cannot silently move it onto another country.
    const [activeCode, setActiveCode] = useState(props.selected.code)
    const activeIndex = Math.max(options.findIndex((v) => v.code === activeCode), 0)
    const activeOption: Value | undefined = options[activeIndex]

    useEffect(() => {
        const active = listRef.current?.querySelector("[data-active=true]")
        // jsdom has no scrollIntoView, and neither does an empty result set.
        active?.scrollIntoView?.({block: "nearest"})
    }, [activeCode, options.length])

    const choose = (value: Value) => {
        setSearch("")
        setActiveCode(value.code)
        props.onChange(value)
    }

    /**
     * Typing puts the cursor on the first actual match, not merely on the first
     * row: `visibleOptions` keeps the selected country in the list even when the
     * search excludes it, and landing on that would make Enter re-pick what is
     * already selected — the keyboard form of the bug #14 fixed for clicks.
     */
    const handleSearchChange = (next: string) => {
        setSearch(next)
        const nextOptions = visibleOptions(props.values, props.selected, next)
        const cursor = next.trim() === ""
            ? props.selected
            : nextOptions.find((v) => matchesSearch(v, next)) ?? props.selected
        setActiveCode(cursor.code)
    }

    /** The keys a `<select size={n}>` used to handle for us. */
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
        // Space only picks inside the list; in the search field it is a space.
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
        <div
            ref={listRef}
            role="listbox"
            tabIndex={0}
            aria-label="Country"
            aria-activedescendant={activeOption && `country-option-${activeOption.code}`}
            onKeyDown={handleListKeyDown}
            className="input-select">
            {options.map((v) => (
                /*
                 * The options are deliberately not focusable and carry no key
                 * handler of their own: the listbox holds the focus and moves a
                 * cursor with aria-activedescendant, which is what the arrow,
                 * Home, End, Enter and Space handling above operates on.
                 */
                // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/interactive-supports-focus
                <div
                    id={`country-option-${v.code}`}
                    className="input-select-option"
                    key={v.code}
                    role="option"
                    aria-selected={v.code === props.selected.code}
                    data-active={v.code === activeOption?.code}
                    onClick={() => choose(v)}>
                    {v.name}
                </div>
            ))}
        </div>
        <input
            type="text"
            placeholder={"🔍 Search..."}
            value={search}
            onChange={(event) => handleSearchChange(event.target.value)}
            onKeyDown={handleSearchKeyDown}
            className="input-search"
            autoComplete="off"
        />
    </div>
}
