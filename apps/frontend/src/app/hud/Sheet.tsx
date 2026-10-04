import {ReactNode, useEffect, useId, useRef} from "react"
import {BackIcon, CloseIcon} from "../components/icons.tsx"
import {useEscape} from "../components/useDialog.ts"
import "./Sheet.css"

export type SheetProps = {
    title: ReactNode
    head?: ReactNode
    onBack?: () => void
    backLabel?: string
    className?: string
    onClose: () => void
    children: ReactNode
}

export default function Sheet(props: SheetProps) {
    const title = useRef<HTMLHeadingElement>(null)
    const titleId = useId()

    useEffect(() => {
        title.current?.focus()
    }, [])

    useEscape(props.onBack ?? props.onClose)

    return <section className={props.className ? `sheet panel ${props.className}` : "sheet panel"}
                    aria-labelledby={titleId}>
        <div className="sheet-head">
            <span className="sheet-handle" aria-hidden="true"/>
            <div className="sheet-title-row">
                {props.onBack && <button type="button"
                                         className="icon-button"
                                         aria-label={props.backLabel ?? "Back"}
                                         onClick={props.onBack}>
                    <BackIcon/>
                </button>}
                <h2 className={props.head ? "sheet-title sr-only" : "sheet-title"}
                    id={titleId}
                    ref={title}
                    tabIndex={-1}>
                    {props.title}
                </h2>
                {props.head}
                <button type="button"
                        className="icon-button sheet-close"
                        aria-label="Close"
                        onClick={props.onClose}>
                    <CloseIcon/>
                </button>
            </div>
        </div>
        <div className="sheet-body">
            {props.children}
        </div>
    </section>
}
