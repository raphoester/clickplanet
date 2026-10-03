import {useEffect, useId, useRef, useState} from "react"
import {Season} from "../../backends/season.ts"
import {finaleLinks} from "../../domain/seasonCalendar.ts"
import {CalendarIcon} from "../components/icons.tsx"
import "./Season.css"

export default function AddToCalendarButton({season}: {season: Season}) {
    const [open, setOpen] = useState(false)
    const shell = useRef<HTMLDivElement>(null)
    const list = useId()

    useEffect(() => {
        if (!open) return
        const pressed = (event: PointerEvent) => {
            if (!shell.current?.contains(event.target as Node)) setOpen(false)
        }
        const typed = (event: KeyboardEvent) => {
            if (event.key === "Escape") setOpen(false)
        }
        document.addEventListener("pointerdown", pressed)
        document.addEventListener("keydown", typed)
        return () => {
            document.removeEventListener("pointerdown", pressed)
            document.removeEventListener("keydown", typed)
        }
    }, [open])

    const links = finaleLinks(season, new URL("/play", window.location.origin).href)

    return <div className="season-calendar" ref={shell}>
        <button type="button"
                className="button button-mini button-secondary"
                aria-expanded={open}
                aria-controls={list}
                onClick={() => setOpen(!open)}>
            <CalendarIcon size={15}/>
            <span>Add to calendar</span>
        </button>

        {open && <ul id={list} className="panel season-calendar-list">
            {links.map(link => <li key={link.name}>
                <a className="season-calendar-link"
                   href={link.url}
                   target={link.web ? "_blank" : undefined}
                   rel={link.web ? "noopener noreferrer" : undefined}
                   onClick={() => setOpen(false)}>
                    {link.name}
                </a>
            </li>)}
        </ul>}
    </div>
}
