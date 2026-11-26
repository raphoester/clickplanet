import {useEffect, useState} from "react"

// A picked file's address while it is shown.
export function usePreview(file: File | undefined): string | undefined {
    const [url, setUrl] = useState<string>()

    useEffect(() => {
        if (!file) {
            setUrl(undefined)
            return
        }
        const made = URL.createObjectURL(file)
        setUrl(made)
        return () => URL.revokeObjectURL(made)
    }, [file])

    return url
}
