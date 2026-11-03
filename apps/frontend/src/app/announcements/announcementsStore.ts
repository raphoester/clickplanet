import {create} from "zustand"
import {Announcement, announced} from "../../domain/announcements.ts"

type AnnouncementsStore = {
    queue: readonly Announcement[]
    announce: (announcement: Announcement) => void
    dismiss: () => void
}

export const useAnnouncements = create<AnnouncementsStore>()((set) => ({
    queue: [],
    announce: (announcement) => set((state) => {
        const queue = announced(state.queue, announcement)
        return queue === state.queue ? state : {queue}
    }),
    dismiss: () => set(({queue}) => ({queue: queue.slice(1)})),
}))
