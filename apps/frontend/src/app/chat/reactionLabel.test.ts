import {describe, expect, it} from "vitest"
import {Reaction} from "../../gen/grpc/chat/v1/chat_pb.ts"
import {ENGLISH} from "../../i18n/translator.ts"
import {reactionLabelOf} from "./reactionLabel.ts"
import {REACTION_IMAGES} from "./reactionsAsset.ts"

describe("reactionLabelOf", () => {
    it("names each reaction by its proto value", () => {
        expect(reactionLabelOf(Reaction.THUMBS_UP)).toBe("reaction.thumbsUp")
        expect(reactionLabelOf(Reaction.LAUGH)).toBe("reaction.laugh")
    })

    it("has a line for every reaction the chat draws", () => {
        expect([...REACTION_IMAGES.keys()].map(reactionLabelOf).filter((key) => !(key in ENGLISH))).toEqual([])
    })
})
