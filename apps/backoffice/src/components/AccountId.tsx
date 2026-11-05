import {useEffect, useState} from "react"

const COPIED_MS = 1500

export function AccountId({id}: {id: string}) {
    if (id === "") return <span className="muted">no account</span>
    return <Copyable text={id}/>
}

// An id or an address, with Copy: what the operator tools take. A short one shows the id's first part, the whole on
// hover, and copies the whole.
export function Copyable({text, short = false}: {text: string, short?: boolean}) {
    const [copied, setCopied] = useState(false)

    useEffect(() => {
        if (!copied) return
        const timer = setTimeout(() => setCopied(false), COPIED_MS)
        return () => clearTimeout(timer)
    }, [copied])

    return (
        <span className="account">
            <code className="mono" title={short ? text : undefined}>{short ? shortOf(text) : text}</code>
            <button
                type="button"
                className="button-small"
                onClick={() => navigator.clipboard.writeText(text).then(() => setCopied(true), () => setCopied(false))}
            >
                {copied ? "Copied" : "Copy"}
            </button>
        </span>
    )
}

// An account id's first group: enough to tell two apart on one page.
function shortOf(text: string): string {
    const [first] = text.split("-")
    return first.length < text.length ? `${first}…` : text
}
