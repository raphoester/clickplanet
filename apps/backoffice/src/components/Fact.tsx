import {ReactNode} from "react"

// One line of a `<dl className="facts">`.
export function Fact({label, children}: {label: string, children: ReactNode}) {
    return (
        <div>
            <dt>{label}</dt>
            <dd>{children}</dd>
        </div>
    )
}
