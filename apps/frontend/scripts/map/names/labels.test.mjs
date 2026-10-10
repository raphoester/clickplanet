import {describe, expect, it} from "vitest"

import {adminName, englishOf, expanded, keyOfName, onlyCompass} from "./labels.mjs"

describe("expanded", () => {
    it("spells out abbreviations and puts capitals back to title case", () => {
        expect(expanded("ZANZIBAR I.")).toBe("Zanzibar Island")
        expect(expanded("Lofoten Is.")).toBe("Lofoten Islands")
        expect(expanded("ISLE OF MAN")).toBe("Isle of Man")
    })
})

describe("englishOf", () => {
    it("keeps the cartographer's label when the English one names something else", () => {
        expect(englishOf("Utupua", "Ryan Bullard")).toBe("Utupua")
    })

    it("keeps the label when the English one only adds a word", () => {
        expect(englishOf("TASMANIA", "Mainland Tasmania")).toBe("Tasmania")
    })

    it("keeps one island from becoming its group", () => {
        expect(englishOf("Zanzibar I.", "Zanzibar Archipelago")).toBe("Zanzibar Island")
    })

    it("takes the English label for the same place", () => {
        expect(englishOf("Corse", "Corsica")).toBe("Corsica")
        expect(englishOf("Føroyar", "Faroe Islands", ["Färöer"])).toBe("Faroe Islands")
    })
})

describe("adminName", () => {
    it("drops the kind of unit", () => {
        expect(adminName({name_en: "Niigata Prefecture"})).toBe("Niigata")
    })

    it("keeps the kind when the rest is only a direction", () => {
        expect(adminName({name_en: "Northern", type_en: "Division"})).toBe("Northern Division")
        expect(onlyCompass("South West")).toBe(true)
        expect(onlyCompass("Western Visayas")).toBe(false)
    })
})

describe("keyOfName", () => {
    it("folds accents, case and punctuation", () => {
        expect(keyOfName("Réunion")).toBe(keyOfName("REUNION"))
        expect(keyOfName("Molokaʻi")).toBe("moloka i")
    })
})
