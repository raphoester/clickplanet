import {describe, expect, it} from "vitest"
import {credited} from "./caption.ts"

describe("the caption of a clip", () => {
    it("ends with the credit its music asks for", () => {
        expect(credited("PALESTINE STRIKES BACK\n#map", "Fida'i by Sounds Monarch, CC BY 3.0"))
            .toBe("PALESTINE STRIKES BACK\n#map\n🎵 Fida'i by Sounds Monarch, CC BY 3.0")
    })

    it("is left as it is when the music asks for none, or the clip plays none", () => {
        expect(credited("PALESTINE STRIKES BACK\n#map", undefined)).toBe("PALESTINE STRIKES BACK\n#map")
    })
})
