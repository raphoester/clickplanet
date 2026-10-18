import {useEffect, useState} from "react"

const COPIED_MS = 1500

export function AccountId({id}: {id: string}) {
    if (id === "") return <span className="muted">no account</span>
    return <Copyable text={id}/>
}

// An id or an address, with Copy: what the operator tools take.
export function Copyable({text}: {text: string}) {
    const [copied, setCopied] = useState(false)

    useEffect(() => {
        if (!copied) return
        const timer = setTimeout(() => setCopied(false), COPIED_MS)
        return () => clearTimeout(timer)
    }, [copied])

    return (
        <span className="account">
            <code className="mono">{text}</code>
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
