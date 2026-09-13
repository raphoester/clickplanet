import {afterEach, beforeEach, describe, expect, it, vi} from "vitest";
import {createAnthemPlayer, FADE_S} from "./anthemPlayer.ts";

type FakeAudio = {url: string, loop: boolean, paused: boolean, play: () => Promise<void>, pause: () => void,
    removeAttribute: () => void, load: () => void, currentTime: number, duration: number}

function fakes() {
    const audios: FakeAudio[] = []
    const param = () => ({
        value: 0,
        setValueAtTime: vi.fn(), linearRampToValueAtTime: vi.fn(),
        cancelScheduledValues: vi.fn(), setTargetAtTime: vi.fn(),
    })
    const node = () => ({gain: param(), connect: vi.fn((next) => next), disconnect: vi.fn()})
    const ctx = {
        state: "running", currentTime: 0, destination: {},
        createGain: vi.fn(node),
        createMediaElementSource: vi.fn(node),
        resume: vi.fn(() => Promise.resolve()),
        close: vi.fn(() => Promise.resolve()),
    }
    const createAudio = (url: string) => {
        const audio: FakeAudio = {
            url, loop: false, paused: true, currentTime: 0, duration: NaN,
            play: () => {
                audio.paused = false
                return Promise.resolve()
            },
            pause: () => {
                audio.paused = true
            },
            removeAttribute: () => {
            },
            load: () => {
            },
        }
        audios.push(audio)
        return audio as unknown as HTMLAudioElement
    }
    const player = createAnthemPlayer({createContext: () => ctx as unknown as AudioContext, createAudio})
    return {player, audios, ctx}
}

describe("createAnthemPlayer", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => vi.useRealTimers())

    it("loads nothing before the first gesture, then plays what it was asked for", () => {
        const {player, audios} = fakes()
        player.setAudible(true)
        player.setTrack("/fr.m4a")
        expect(audios).toHaveLength(0)

        player.unlock()
        expect(audios.map((a) => a.url)).toEqual(["/fr.m4a"])
        expect(audios[0].loop).toBe(true)
        expect(audios[0].paused).toBe(false)
    })

    it("crossfades: the old recording stops only once its fade is over", () => {
        const {player, audios} = fakes()
        player.setAudible(true)
        player.unlock()
        player.setTrack("/fr.m4a")
        player.setTrack("/de.m4a")

        expect(audios.map((a) => a.url)).toEqual(["/fr.m4a", "/de.m4a"])
        expect(audios[0].paused).toBe(false)
        vi.advanceTimersByTime(FADE_S * 1000 + 200)
        expect(audios[0].paused).toBe(true)
        expect(audios[1].paused).toBe(false)
    })

    it("does not restart a recording asked for again", () => {
        const {player, audios} = fakes()
        player.unlock()
        player.setTrack("/fr.m4a")
        player.setTrack("/fr.m4a")
        expect(audios).toHaveLength(1)
    })

    it("pauses when it goes quiet and resumes when it is heard again", () => {
        const {player, audios} = fakes()
        player.setAudible(true)
        player.unlock()
        player.setTrack("/fr.m4a")

        player.setAudible(false)
        vi.advanceTimersByTime(1100)
        expect(audios[0].paused).toBe(true)

        player.setAudible(true)
        expect(audios[0].paused).toBe(false)
    })

    it("stays paused when muted before the first gesture", () => {
        const {player, audios} = fakes()
        player.setTrack("/fr.m4a")
        player.unlock()
        expect(audios[0].paused).toBe(true)
    })
})
