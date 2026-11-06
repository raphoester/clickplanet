import {afterEach, describe, expect, it} from "vitest"
import {ChatMessage} from "../../backends/chat.ts"
import {useChat} from "./chatStore.ts"

const message = (id: string): ChatMessage =>
    ({id, sentAt: 0, authorAccountId: "account-ana", countryCode: "fr", text: "hi", reactions: [], reactionsVersion: 0})

afterEach(() => useChat.setState(useChat.getInitialState(), true))

describe("the chat", () => {
    it("counts nothing unread until the panel says so", () => {
        expect(useChat.getState().unread).toBe(0)

        useChat.getState().setUnread(4)

        expect(useChat.getState().unread).toBe(4)
    })

    it("keeps every message reported", () => {
        const {markReported} = useChat.getState()

        markReported(message("1"))
        markReported(message("2"))

        expect([...useChat.getState().reported]).toEqual(["1", "2"])
    })
})
