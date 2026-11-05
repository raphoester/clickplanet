import {create} from "zustand"
import {ChatMessage} from "../../backends/chat.ts"

type ChatStore = {
    // What the player has not seen yet, in both rooms; the phone's Chat tab wears it.
    unread: number
    // The messages this page reported.
    reported: ReadonlySet<string>
    setUnread: (unread: number) => void
    markReported: (message: ChatMessage) => void
}

const NOTHING: ReadonlySet<string> = new Set()

export const useChat = create<ChatStore>()((set) => ({
    unread: 0,
    reported: NOTHING,
    setUnread: (unread) => set({unread}),
    markReported: (message) => set(({reported}) => ({reported: new Set([...reported, message.id])})),
}))
