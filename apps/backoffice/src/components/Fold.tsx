import {ReactNode, useId, useState} from "react"

type Props = {
    label: string
    title: ReactNode
    // A word beside the title while it is shut: how much is inside.
    sub?: ReactNode
    open?: boolean
    children: ReactNode
}

// A panel shut by default under its title. What is inside renders, and so reads, only once it is opened.
export function Fold({label, title, sub, open: first = false, children}: Props) {
    const [open, setOpen] = useState(first)
    const body = useId()

    return (
        <section className="panel fold" aria-label={label}>
            <h2>
                <button
                    type="button"
                    className="fold-head"
                    aria-expanded={open}
                    aria-controls={body}
                    onClick={() => setOpen(!open)}
                >
                    <span className="fold-mark" aria-hidden="true">{open ? "▾" : "▸"}</span>
                    {title}
                    {sub !== undefined && <span className="muted fold-sub">{sub}</span>}
                </button>
            </h2>
            {open && <div id={body} className="fold-body">{children}</div>}
        </section>
    )
}
