import {describe, expect, it} from "vitest"
import {ENGLISH} from "../../i18n/translator.ts"
import {reactionLabelOf} from "./reactionLabel.ts"
import {REACTION_IMAGES} from "./reactionsAsset.ts"

describe("reactionLabelOf", () => {
    it("has a line for every reaction the chat draws", () => {
        expect([...REACTION_IMAGES.keys()].map(reactionLabelOf).filter((key) => !(key in ENGLISH))).toEqual([])
    })
})
