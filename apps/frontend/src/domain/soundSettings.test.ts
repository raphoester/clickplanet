import {describe, expect, it} from "vitest"
import {DEFAULT_SOUND_SETTINGS, isAudible, parseSoundSettings} from "./soundSettings.ts"

describe("parseSoundSettings", () => {
    it("starts with every sound on", () => {
        expect(parseSoundSettings(null)).toEqual(DEFAULT_SOUND_SETTINGS)
        expect(Object.values(DEFAULT_SOUND_SETTINGS.sounds).every(Boolean)).toBe(true)
    })

    it("reads back what was saved", () => {
        const saved = {...DEFAULT_SOUND_SETTINGS, sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false}}
        expect(parseSoundSettings(JSON.stringify(saved))).toEqual(saved)
    })

    it("falls back to the default for anything it cannot read", () => {
        expect(parseSoundSettings("not json")).toEqual(DEFAULT_SOUND_SETTINGS)
        expect(parseSoundSettings("42")).toEqual(DEFAULT_SOUND_SETTINGS)
        expect(parseSoundSettings(JSON.stringify({enabled: "yes", sounds: {click: 1}}))).toEqual(DEFAULT_SOUND_SETTINGS)
    })

    it("turns on a sound the saved settings do not know about", () => {
        const parsed = parseSoundSettings(JSON.stringify({enabled: false, sounds: {click: false}}))
        expect(parsed.enabled).toBe(false)
        expect(parsed.sounds.click).toBe(false)
        expect(parsed.sounds.bomb).toBe(true)
    })
})

describe("isAudible", () => {
    it("needs both the master switch and the sound's own", () => {
        const chatOff = {enabled: true, sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false}}
        expect(isAudible(chatOff, "click")).toBe(true)
        expect(isAudible(chatOff, "chat")).toBe(false)
        expect(isAudible({...DEFAULT_SOUND_SETTINGS, enabled: false}, "click")).toBe(false)
    })

    it("lets the click switch cover every bonus click", () => {
        const clickOff = {enabled: true, sounds: {...DEFAULT_SOUND_SETTINGS.sounds, click: false}}
        for (const name of ["spread", "boost", "enclose"] as const) {
            expect(isAudible(DEFAULT_SOUND_SETTINGS, name)).toBe(true)
            expect(isAudible(clickOff, name)).toBe(false)
        }
    })
})
