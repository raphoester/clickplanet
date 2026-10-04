import {useEffect, useState} from "react"

export function usePageVisible(): boolean {
    const [visible, setVisible] = useState(() => !document.hidden)

    useEffect(() => {
        const onVisibility = () => setVisible(!document.hidden)
        document.addEventListener("visibilitychange", onVisibility)
        return () => document.removeEventListener("visibilitychange", onVisibility)
    }, [])

    return visible
}
