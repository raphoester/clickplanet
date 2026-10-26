import {useCallback, useState} from "react"
import {Announcement, announced} from "../../domain/announcements.ts"

export type Announcements = {
    current: Announcement | undefined
    announce: (announcement: Announcement) => void
    dismiss: () => void
}

export function useAnnouncements(): Announcements {
    const [queue, setQueue] = useState<readonly Announcement[]>([])

    const announce = useCallback((announcement: Announcement) => setQueue((waiting) => announced(waiting, announcement)), [])
    const dismiss = useCallback(() => setQueue((waiting) => waiting.slice(1)), [])

    return {current: queue[0], announce, dismiss}
}
