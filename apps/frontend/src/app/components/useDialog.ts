import {RefObject, useEffect, useRef} from "react";

const FOCUSABLE = 'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])'

const focusableIn = (container: HTMLElement | null): HTMLElement[] =>
    Array.from(container?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? [])
        .filter((el) => !el.hasAttribute("disabled"))

function useLatest<T>(value: T) {
    const ref = useRef(value)
    ref.current = value
    return ref
}

export function useEscape(onClose: () => void) {
    const latest = useLatest(onClose)

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key !== "Escape") return
            event.preventDefault()
            latest.current()
        }
        document.addEventListener("keydown", onKeyDown)
        return () => document.removeEventListener("keydown", onKeyDown)
    }, [latest])
}

export function useModalDialog(panel: RefObject<HTMLElement>, onClose: () => void) {
    const latest = useLatest(onClose)

    useEffect(() => {
        const opener = document.activeElement as HTMLElement | null

        const first = focusableIn(panel.current)[0]
        if (first) first.focus()
        else panel.current?.focus()

        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                event.preventDefault()
                latest.current()
                return
            }
            if (event.key !== "Tab") return

            const items = focusableIn(panel.current)
            if (items.length === 0) return

            const edge = event.shiftKey ? items[0] : items[items.length - 1]
            const wrapTo = event.shiftKey ? items[items.length - 1] : items[0]
            const inside = panel.current?.contains(document.activeElement)

            if (document.activeElement === edge || !inside) {
                event.preventDefault()
                wrapTo.focus()
            }
        }

        document.addEventListener("keydown", onKeyDown)
        return () => {
            document.removeEventListener("keydown", onKeyDown)
            opener?.focus?.()
        }
    }, [panel, latest])
}
