import {useCallback, useEffect, useRef, useState} from "react"
import {MENU, PageId} from "../navigation.ts"

type Props = {page: PageId, onGo: (page: PageId) => void}

// The burger at the top left, and the drawer it opens from the left with every page, by group. The drawer is
// modal: Escape, a click beside it or a page picked closes it, and the focus goes back to the burger.
export function Menu({page, onGo}: Props) {
    const [open, setOpen] = useState(false)
    const burger = useRef<HTMLButtonElement>(null)
    const drawer = useRef<HTMLDivElement>(null)

    const close = useCallback(() => {
        setOpen(false)
        burger.current?.focus()
    }, [])

    useEffect(() => {
        if (!open) return
        drawer.current?.querySelector<HTMLElement>("[aria-current=page]")?.focus()

        const onKey = (event: KeyboardEvent) => {
            if (event.key === "Escape") close()
            if (event.key === "Tab") trapFocus(event, drawer.current)
        }
        const onPointer = (event: PointerEvent) => {
            if (!drawer.current?.contains(event.target as Node)) close()
        }
        // The page behind does not scroll under the drawer.
        const overflow = document.body.style.overflow
        document.body.style.overflow = "hidden"
        document.addEventListener("keydown", onKey)
        document.addEventListener("pointerdown", onPointer)
        return () => {
            document.body.style.overflow = overflow
            document.removeEventListener("keydown", onKey)
            document.removeEventListener("pointerdown", onPointer)
        }
    }, [open, close])

    const go = (to: PageId) => {
        onGo(to)
        close()
    }

    return (
        <>
            <button
                ref={burger}
                type="button"
                className="menu-button"
                aria-label="Menu"
                aria-expanded={open}
                aria-controls="menu"
                onClick={() => setOpen(true)}
            >
                <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true">
                    <path d="M3 6h18M3 12h18M3 18h18" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
                </svg>
            </button>
            {open && (
                <>
                    <div className="drawer-backdrop"/>
                    <div
                        ref={drawer}
                        id="menu"
                        className="drawer"
                        role="dialog"
                        aria-modal="true"
                        aria-label="Menu"
                    >
                        <div className="drawer-head">
                            <span className="drawer-title">Clickplanet backoffice</span>
                            <button type="button" className="menu-close" aria-label="Close the menu" onClick={close}>
                                <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
                                    <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
                                </svg>
                            </button>
                        </div>
                        <nav aria-label="Pages">
                            {MENU.map(group => (
                                <div key={group.label} className="menu-group">
                                    <h2 className="menu-heading">{group.label}</h2>
                                    <ul>
                                        {group.pages.map(item => (
                                            <li key={item.page}>
                                                <button
                                                    type="button"
                                                    className="menu-item"
                                                    aria-current={item.page === page ? "page" : undefined}
                                                    onClick={() => go(item.page)}
                                                >
                                                    {item.label}
                                                </button>
                                            </li>
                                        ))}
                                    </ul>
                                </div>
                            ))}
                        </nav>
                    </div>
                </>
            )}
        </>
    )
}

// Tab from the last control comes back to the first, and Shift+Tab from the first to the last: the page behind
// the drawer cannot be reached while it is open.
function trapFocus(event: KeyboardEvent, drawer: HTMLElement | null) {
    if (drawer === null) return
    const controls = [...drawer.querySelectorAll<HTMLElement>("button:not(:disabled)")]
    const first = controls[0]
    const last = controls[controls.length - 1]
    if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
    }
}
