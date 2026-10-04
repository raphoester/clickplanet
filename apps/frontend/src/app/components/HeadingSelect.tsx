import {KeyboardEvent, ReactNode, useEffect, useId, useLayoutEffect, useRef, useState} from "react"
import {ChevronIcon} from "./icons.tsx"
import "./HeadingSelect.css"

export type HeadingChoice<T extends string> = {
    value: T
    name: string
    label: ReactNode
}

const EDGE = 8

type Place = {top: number, left: number}

type HeadingSelectProps<T extends string> = {
    label: string
    choices: readonly HeadingChoice<T>[]
    value: T
    onChange: (value: T) => void
}

export default function HeadingSelect<T extends string>({label, choices, value, onChange}: HeadingSelectProps<T>) {
    const [open, setOpen] = useState(false)
    const [active, setActive] = useState(value)
    const [place, setPlace] = useState<Place>()
    const id = useId()
    const root = useRef<HTMLDivElement>(null)
    const button = useRef<HTMLButtonElement>(null)
    const list = useRef<HTMLDivElement>(null)

    const current = choices.find((choice) => choice.value === value) ?? choices[0]
    const activeIndex = Math.max(choices.findIndex((choice) => choice.value === active), 0)

    useLayoutEffect(() => {
        if (!open) return
        let frame = 0
        const follow = () => {
            const next = placeOf(button.current, list.current)
            setPlace((current) => next && (next.top !== current?.top || next.left !== current?.left) ? next : current)
            frame = requestAnimationFrame(follow)
        }
        follow()
        return () => cancelAnimationFrame(frame)
    }, [open])

    useEffect(() => {
        if (!open) return
        list.current?.focus({preventScroll: true})
        const away = (event: PointerEvent) => {
            if (!root.current?.contains(event.target as Node)) setOpen(false)
        }
        const moved = (event: Event) => {
            if (event.target instanceof Node && event.target.contains(root.current)) setOpen(false)
        }
        const resized = () => setOpen(false)
        document.addEventListener("pointerdown", away)
        document.addEventListener("scroll", moved, true)
        window.addEventListener("resize", resized)
        return () => {
            document.removeEventListener("pointerdown", away)
            document.removeEventListener("scroll", moved, true)
            window.removeEventListener("resize", resized)
        }
    }, [open])

    const show = () => {
        setActive(value)
        setOpen(true)
    }

    const close = () => {
        setOpen(false)
        button.current?.focus({preventScroll: true})
    }

    const choose = (chosen: T) => {
        onChange(chosen)
        close()
    }

    const onButtonKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
        if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return
        event.preventDefault()
        show()
    }

    const onListKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
        const moveTo = (index: number) => setActive(choices[index].value)
        switch (event.key) {
            case "ArrowDown":
                moveTo(Math.min(activeIndex + 1, choices.length - 1))
                break
            case "ArrowUp":
                moveTo(Math.max(activeIndex - 1, 0))
                break
            case "Home":
                moveTo(0)
                break
            case "End":
                moveTo(choices.length - 1)
                break
            case "Enter":
            case " ":
                choose(choices[activeIndex].value)
                break
            case "Escape":
                close()
                break
            case "Tab":
                setOpen(false)
                return
            default:
                return
        }
        event.preventDefault()
    }

    return <div className="heading-select" ref={root}>
        <button ref={button}
                type="button"
                className="menu-section-title heading-select-button"
                aria-label={`${label}: ${current.name}`}
                aria-haspopup="listbox"
                aria-expanded={open}
                aria-controls={open ? `${id}-list` : undefined}
                onClick={() => open ? close() : show()}
                onKeyDown={onButtonKeyDown}>
            {current.label}
            <ChevronIcon/>
        </button>
        {open && <div ref={list}
                     id={`${id}-list`}
                     role="listbox"
                     tabIndex={-1}
                     aria-label={label}
                     aria-activedescendant={`${id}-${choices[activeIndex].value}`}
                     className="panel-box heading-select-list"
                     style={place}
                     onKeyDown={onListKeyDown}>
            {choices.map((choice) => (
                // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/interactive-supports-focus
                <div key={choice.value}
                     id={`${id}-${choice.value}`}
                     role="option"
                     aria-selected={choice.value === value}
                     data-active={choice.value === choices[activeIndex].value}
                     className="heading-select-option"
                     onClick={() => choose(choice.value)}>
                    {choice.label}
                </div>
            ))}
        </div>}
    </div>
}

function placeOf(button: HTMLElement | null, list: HTMLElement | null): Place | undefined {
    if (!button || !list) return undefined
    const at = button.getBoundingClientRect()
    const height = list.offsetHeight
    const below = at.bottom + height <= window.innerHeight - EDGE || at.top - height < EDGE
    return {top: below ? at.bottom : at.top - height, left: at.left}
}
