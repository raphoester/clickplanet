import {afterEach, describe, expect, it} from "vitest"
import {StoryKind, VideoStory} from "../../domain/video/story.ts"
import {speak} from "../../i18n/page.ts"
import {ENGLISH_TRANSLATOR, Translator} from "../../i18n/translator.ts"
import fr from "../../i18n/locales/fr.json"
import {videoHeadline} from "./videoHeadline.ts"

afterEach(() => speak(ENGLISH_TRANSLATOR))

const story = (kind: StoryKind, start: number, end: number): VideoStory => ({flag: "pl", country: "de", kind, tiles: 100, start, end})

describe("videoHeadline", () => {
    it("says the flag's day in capitals, as the video does", () => {
        expect(videoHeadline(story("gain", 0, 40))).toBe("POLAND TOOK 40% OF GERMANY")
        expect(videoHeadline(story("loss", 60, 10))).toBe("POLAND IS LOSING GERMANY. HELP")
        expect(videoHeadline(story("quiet", 62, 62))).toBe("POLAND HOLDS 62% OF GERMANY")
    })

    it("does not name the country twice when the flag's day is in its own", () => {
        const home = (kind: StoryKind, start: number, end: number) => ({...story(kind, start, end), country: "pl"})

        expect(videoHeadline(home("gain", 0, 40))).toBe("POLAND TOOK BACK 40% OF ITS LAND")
        expect(videoHeadline(home("loss", 60, 10))).toBe("POLAND IS LOSING GROUND AT HOME. HELP")
        expect(videoHeadline(home("quiet", 62, 62))).toBe("POLAND HOLDS 62% OF ITS LAND")
    })

    it("says it in the player's language", () => {
        speak(new Translator("fr", fr, "fr-FR"))

        expect(videoHeadline({...story("loss", 60, 10), flag: "fr", country: "fr"})).toBe("FRANCE PERD DU TERRAIN À DOMICILE. À L’AIDE")
    })
})
