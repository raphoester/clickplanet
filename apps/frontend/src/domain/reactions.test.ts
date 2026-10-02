import {describe, expect, it} from "vitest";
import {ChatMessage, Reaction, ReactionCount} from "../backends/chat.ts";
import {
    applyReactionsAnswer,
    applyReactionsChange,
    mergedReactions,
    toggledReactions,
    whoReacted,
    withReactions,
} from "./reactions.ts";

function message(id: string, reactions: ChatMessage["reactions"] = []): ChatMessage {
    return {
        id, sentAt: 0, authorName: "Ana", authorAdmin: false,
        countryCode: "fr", text: "hi", reactions, reactionsVersion: 1,
    }
}

function count(reaction: Reaction, count: number, mine: boolean, reactors: string[] = []): ReactionCount {
    return {reaction, count, mine, reactors}
}

describe("mergedReactions", () => {
    it("keeps what the log knew about mine, since the stream knows nobody", () => {
        expect(mergedReactions(
            [count(Reaction.CLOWN, 1, true)],
            [count(Reaction.CLOWN, 2, false), count(Reaction.SKULL, 1, false)],
        )).toEqual([
            count(Reaction.CLOWN, 2, true),
            count(Reaction.SKULL, 1, false),
        ])
    })

    it("takes who reacted from the stream, which does know that", () => {
        expect(mergedReactions(
            [count(Reaction.CLOWN, 1, true, ["Ana"])],
            [count(Reaction.CLOWN, 2, false, ["Ana", "Bo"])],
        )).toEqual([count(Reaction.CLOWN, 2, true, ["Ana", "Bo"])])
    })

    it("drops a reaction the stream no longer carries", () => {
        expect(mergedReactions([count(Reaction.CLOWN, 1, true)], [])).toEqual([])
    })
})

describe("toggledReactions", () => {
    it("puts a new reaction last, under this player's name", () => {
        expect(toggledReactions([count(Reaction.CLOWN, 2, false)], Reaction.FIRE, true, "Ana")).toEqual([
            count(Reaction.CLOWN, 2, false),
            count(Reaction.FIRE, 1, true, ["Ana"]),
        ])
    })

    it("joins a reaction others gave, and is named last of them", () => {
        expect(toggledReactions([count(Reaction.CLOWN, 2, false, ["Bo", "Cy"])], Reaction.CLOWN, true, "Ana"))
            .toEqual([count(Reaction.CLOWN, 3, true, ["Bo", "Cy", "Ana"])])
    })

    it("takes one off with its name, and the reaction with it when nobody is left", () => {
        expect(toggledReactions([count(Reaction.CLOWN, 2, true, ["Bo", "Ana"])], Reaction.CLOWN, false, "Ana"))
            .toEqual([count(Reaction.CLOWN, 1, false, ["Bo"])])
        expect(toggledReactions([count(Reaction.CLOWN, 1, true, ["Ana"])], Reaction.CLOWN, false, "Ana"))
            .toEqual([])
    })

    it("counts a player it cannot name, and names nobody new", () => {
        expect(toggledReactions([count(Reaction.CLOWN, 1, false, ["Bo"])], Reaction.CLOWN, true))
            .toEqual([count(Reaction.CLOWN, 2, true, ["Bo"])])
        expect(toggledReactions([], Reaction.CLOWN, true)).toEqual([count(Reaction.CLOWN, 1, true)])
    })

    it("changes nothing when asked for what is there", () => {
        const counts = [count(Reaction.CLOWN, 1, true, ["Ana"])]
        expect(toggledReactions(counts, Reaction.CLOWN, true, "Ana")).toEqual(counts)
        expect(toggledReactions(counts, Reaction.SKULL, false, "Ana")).toEqual(counts)
    })
})

describe("whoReacted", () => {
    it("names everyone it has", () => {
        expect(whoReacted(count(Reaction.HEART, 2, false, ["Ana", "Bo"])))
            .toEqual({names: ["Ana", "Bo"], more: 0})
    })

    it("counts the rest of a list the server cut", () => {
        expect(whoReacted(count(Reaction.HEART, 30, false, ["Ana", "Bo"])))
            .toEqual({names: ["Ana", "Bo"], more: 28})
    })

    it("names nobody on a reaction from before names were kept", () => {
        expect(whoReacted(count(Reaction.HEART, 3, false))).toEqual({names: [], more: 3})
    })

    it("lets the count decide, so a name past it is not shown", () => {
        expect(whoReacted(count(Reaction.HEART, 1, false, ["Ana", "Bo"])))
            .toEqual({names: ["Ana"], more: 0})
    })
})

describe("applyReactionsChange", () => {
    it("replaces one message's reactions and leaves the others", () => {
        const log = [message("a"), message("b", [count(Reaction.HEART, 1, true)])]

        const next = applyReactionsChange(log, {
            messageId: "b",
            reactions: [count(Reaction.HEART, 2, false)],
            version: 2,
        })

        expect(next[0]).toBe(log[0])
        expect(next[1].reactions).toEqual([count(Reaction.HEART, 2, true)])
        expect(next[1].reactionsVersion).toBe(2)
    })

    it("drops a frame older than what the log holds, and takes one as new", () => {
        const log = [message("a", [count(Reaction.HEART, 3, false)])]
        const stale = {messageId: "a", reactions: [], version: 0}

        expect(applyReactionsChange(log, stale)).toBe(log)
        expect(applyReactionsAnswer(log, stale)).toBe(log)
        expect(applyReactionsChange(log, {...stale, version: 1})[0].reactions).toEqual([])
    })

    it("takes the answer to this player's own reaction as it is, mine included", () => {
        const log = [message("a", [count(Reaction.HEART, 1, false)])]

        expect(applyReactionsAnswer(log, {
            messageId: "a",
            reactions: [count(Reaction.HEART, 1, true, ["Ana"])],
            version: 2,
        })[0].reactions).toEqual([count(Reaction.HEART, 1, true, ["Ana"])])
    })

    it("ignores a message the log does not hold", () => {
        const log = [message("a")]

        expect(withReactions(log, "gone", () => [count(Reaction.HEART, 1, false)])).toBe(log)
    })
})
