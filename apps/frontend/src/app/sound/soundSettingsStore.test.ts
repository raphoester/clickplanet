// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {DEFAULT_SOUND_SETTINGS, SOUND_SETTINGS_STORAGE_KEY} from "../../domain/soundSettings.ts"

const DEV_KEY = `${SOUND_SETTINGS_STORAGE_KEY}-dev`

afterEach(() => {
    window.localStorage.clear()
    vi.resetModules()
})

const loaded = async () => (await import("./soundSettingsStore.ts")).useSoundSettings

describe("the sound settings", () => {
    it("start from what this browser saved", async () => {
        const saved = {...DEFAULT_SOUND_SETTINGS, sounds: {...DEFAULT_SOUND_SETTINGS.sounds, chat: false}}
        window.localStorage.setItem(DEV_KEY, JSON.stringify(saved))

        expect((await loaded()).getState().settings).toEqual(saved)
    })

    it("save each change in this browser", async () => {
        const sound = await loaded()

        sound.getState().setSettings(DEFAULT_SOUND_SETTINGS)

        expect(JSON.parse(window.localStorage.getItem(DEV_KEY)!)).toEqual(DEFAULT_SOUND_SETTINGS)
    })
})
