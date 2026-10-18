import {useEffect, useState} from "react"

const COPIED_MS = 1500

export function AccountId({id}: {id: string}) {
    const [copied, setCopied] = useState(false)

    useEffect(() => {
        if (!copied) return
        const timer = setTimeout(() => setCopied(false), COPIED_MS)
        return () => clearTimeout(timer)
    }, [copied])

    if (id === "") return <span className="muted">no account</span>

    return (
        <span className="account">
            <code className="mono">{id}</code>
            <button
                type="button"
                className="button-small"
                onClick={() => navigator.clipboard.writeText(id).then(() => setCopied(true), () => setCopied(false))}
            >
                {copied ? "Copied" : "Copy"}
            </button>
        </span>
    )
}
