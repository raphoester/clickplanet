import {ReactNode, useId, useState} from "react"
import {ChevronIcon} from "./icons.tsx"
import "./Fold.css"

export type FoldProps = {
    title: string
    summary?: ReactNode
    summaryLabel?: string
    startOpen?: boolean
    children: ReactNode
}

export default function Fold({title, summary, summaryLabel, startOpen = false, children}: FoldProps) {
    const [open, setOpen] = useState(startOpen)
    const titleId = useId()
    const bodyId = useId()

    return <section className="fold" aria-labelledby={titleId}>
        <h3 className="fold-heading">
            <button type="button"
                    className="panel-box fold-bar"
                    aria-expanded={open}
                    aria-label={summaryLabel && `${title}, ${summaryLabel}`}
                    aria-controls={open ? bodyId : undefined}
                    onClick={() => setOpen((was) => !was)}>
                <span className="fold-title" id={titleId}>{title}</span>
                {summary}
                <span className="fold-chevron"><ChevronIcon size={16}/></span>
            </button>
        </h3>
        {open && <div className="fold-body" id={bodyId}>{children}</div>}
    </section>
}
