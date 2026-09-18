import {describe, expect, it} from "vitest";
import {ChatMessage, Reaction} from "../backends/chat.ts";
import {
    applyReactionsAnswer,
    applyReactionsChange,
    mergedReactions,
    toggledReactions,
    withReactions,
} from "./reactions.ts";

function message(id: string, reactions: ChatMessage["reactions"] = []): ChatMessage {
    return {
        id, sentAt: 0, authorName: "Ana", authorAdmin: false,
        countryCode: "fr", text: "hi", reactions, reactionsVersion: 1,
    }
}

describe("mergedReactions", () => {
    it("keeps what the log knew about mine, since the stream knows nobody", () => {
        expect(mergedReactions(
            [{reaction: Reaction.CLOWN, count: 1, mine: true}],
            [{reaction: Reaction.CLOWN, count: 2, mine: false}, {reaction: Reaction.SKULL, count: 1, mine: false}],
        )).toEqual([
            {reaction: Reaction.CLOWN, count: 2, mine: true},
            {reaction: Reaction.SKULL, count: 1, mine: false},
        ])
    })

    it("drops a reaction the stream no longer carries", () => {
        expect(mergedReactions([{reaction: Reaction.CLOWN, count: 1, mine: true}], [])).toEqual([])
    })
})

describe("toggledReactions", () => {
    it("puts a new reaction last", () => {
        expect(toggledReactions([{reaction: Reaction.CLOWN, count: 2, mine: false}], Reaction.FIRE, true)).toEqual([
            {reaction: Reaction.CLOWN, count: 2, mine: false},
            {reaction: Reaction.FIRE, count: 1, mine: true},
        ])
    })

    it("joins a reaction others gave", () => {
        expect(toggledReactions([{reaction: Reaction.CLOWN, count: 2, mine: false}], Reaction.CLOWN, true))
            .toEqual([{reaction: Reaction.CLOWN, count: 3, mine: true}])
    })

    it("takes one off, and the reaction with it when nobody is left", () => {
        expect(toggledReactions([{reaction: Reaction.CLOWN, count: 2, mine: true}], Reaction.CLOWN, false))
            .toEqual([{reaction: Reaction.CLOWN, count: 1, mine: false}])
        expect(toggledReactions([{reaction: Reaction.CLOWN, count: 1, mine: true}], Reaction.CLOWN, false))
            .toEqual([])
    })

    it("changes nothing when asked for what is there", () => {
        const counts = [{reaction: Reaction.CLOWN, count: 1, mine: true}]
        expect(toggledReactions(counts, Reaction.CLOWN, true)).toEqual(counts)
        expect(toggledReactions(counts, Reaction.SKULL, false)).toEqual(counts)
    })
})

describe("applyReactionsChange", () => {
    it("replaces one message's reactions and leaves the others", () => {
        const log = [message("a"), message("b", [{reaction: Reaction.HEART, count: 1, mine: true}])]

        const next = applyReactionsChange(log, {
            messageId: "b",
            reactions: [{reaction: Reaction.HEART, count: 2, mine: false}],
            version: 2,
        })

        expect(next[0]).toBe(log[0])
        expect(next[1].reactions).toEqual([{reaction: Reaction.HEART, count: 2, mine: true}])
        expect(next[1].reactionsVersion).toBe(2)
    })

    it("drops a frame older than what the log holds, and takes one as new", () => {
        const log = [message("a", [{reaction: Reaction.HEART, count: 3, mine: false}])]
        const stale = {messageId: "a", reactions: [], version: 0}

        expect(applyReactionsChange(log, stale)).toBe(log)
        expect(applyReactionsAnswer(log, stale)).toBe(log)
        expect(applyReactionsChange(log, {...stale, version: 1})[0].reactions).toEqual([])
    })

    it("takes the answer to this player's own reaction as it is, mine included", () => {
        const log = [message("a", [{reaction: Reaction.HEART, count: 1, mine: false}])]

        expect(applyReactionsAnswer(log, {
            messageId: "a",
            reactions: [{reaction: Reaction.HEART, count: 1, mine: true}],
            version: 2,
        })[0].reactions).toEqual([{reaction: Reaction.HEART, count: 1, mine: true}])
    })

    it("ignores a message the log does not hold", () => {
        const log = [message("a")]

        expect(withReactions(log, "gone", () => [{reaction: Reaction.HEART, count: 1, mine: false}])).toBe(log)
    })
})
