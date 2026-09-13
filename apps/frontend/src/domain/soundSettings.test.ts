import {describe, expect, it} from "vitest"
import {DEFAULT_SOUND_SETTINGS, isAnthemAudible, isAudible, parseSoundSettings} from "./soundSettings.ts"

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

describe("the anthem settings", () => {
    it("start on, quietly", () => {
        expect(DEFAULT_SOUND_SETTINGS.anthem).toEqual({on: true, volume: 0.3})
    })

    it("read back what was saved", () => {
        const saved = {...DEFAULT_SOUND_SETTINGS, anthem: {on: false, volume: 0.8}}
        expect(parseSoundSettings(JSON.stringify(saved))).toEqual(saved)
    })

    it("fall back field by field", () => {
        const stored = {enabled: true, sounds: {}, anthem: {on: false, volume: 7}}
        expect(parseSoundSettings(JSON.stringify(stored)).anthem).toEqual({on: false, volume: 0.3})
        // Settings saved before the anthem existed.
        expect(parseSoundSettings(JSON.stringify({enabled: true, sounds: {}})).anthem).toEqual({on: true, volume: 0.3})
    })

    it("are audible only with the master switch on and some volume", () => {
        expect(isAnthemAudible(DEFAULT_SOUND_SETTINGS)).toBe(true)
        expect(isAnthemAudible({...DEFAULT_SOUND_SETTINGS, enabled: false})).toBe(false)
        expect(isAnthemAudible({...DEFAULT_SOUND_SETTINGS, anthem: {on: true, volume: 0}})).toBe(false)
        expect(isAnthemAudible({...DEFAULT_SOUND_SETTINGS, anthem: {on: false, volume: 1}})).toBe(false)
    })
})

describe("isAudible", () => {
    it("needs both the master switch and the sound's own", () => {
        const chatOff = {...DEFAULT_SOUND_SETTINGS, sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false}}
        expect(isAudible(chatOff, "click")).toBe(true)
        expect(isAudible(chatOff, "chat")).toBe(false)
        expect(isAudible({...DEFAULT_SOUND_SETTINGS, enabled: false}, "click")).toBe(false)
    })
})
