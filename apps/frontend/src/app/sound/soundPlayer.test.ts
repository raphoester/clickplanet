import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {createSoundPlayer, MIN_GAP_MS, SILENCE_MS} from "./soundPlayer.ts"
import {SoundName, SOUNDS} from "../../domain/soundSettings.ts"
import {Synth} from "./synths.ts"

function setup(overrides: {audible?: (name: SoundName) => boolean, hidden?: boolean, state?: AudioContextState} = {}) {
    let clock = 1000
    const context = {
        state: overrides.state ?? "running",
        currentTime: 3,
        resume: vi.fn(async () => {
            context.state = "running"
        }),
        suspend: vi.fn(async () => {
            context.state = "suspended"
        }),
        close: vi.fn(async () => {
        }),
    }
    const createContext = vi.fn(() => context as unknown as AudioContext)
    const synths = Object.fromEntries(SOUNDS.map((name) => [name, vi.fn<Synth>()])) as Record<SoundName, ReturnType<typeof vi.fn<Synth>>>
    const player = createSoundPlayer({
        isAudible: overrides.audible ?? (() => true),
        createContext,
        synths,
        now: () => clock,
        isHidden: () => overrides.hidden ?? false,
    })
    return {player, synths, context, createContext, advance: (ms: number) => clock += ms}
}

describe("createSoundPlayer", () => {
    beforeEach(() => vi.useFakeTimers())
    afterEach(() => vi.useRealTimers())

    it("stays silent until a gesture has unlocked it", () => {
        const {player, synths} = setup()
        player.play("click")
        expect(synths.click).not.toHaveBeenCalled()

        player.unlock()
        player.play("click")
        expect(synths.click).toHaveBeenCalledWith(expect.anything(), 3, {volume: 1, onWater: false})
    })

    it("plays nothing the settings have switched off", () => {
        const {player, synths} = setup({audible: (name) => name !== "chat"})
        player.unlock()
        player.play("chat")
        player.play("bomb", {volume: 0.5, onWater: true})
        expect(synths.chat).not.toHaveBeenCalled()
        expect(synths.bomb).toHaveBeenCalledWith(expect.anything(), 3, {volume: 0.5, onWater: true})
    })

    it("plays nothing in a hidden tab", () => {
        const {player, synths} = setup({hidden: true})
        player.unlock()
        player.play("bonusSpawn")
        expect(synths.bonusSpawn).not.toHaveBeenCalled()
    })

    it("drops a replay that comes too soon, per sound", () => {
        const {player, synths, advance} = setup()
        player.unlock()

        player.play("chat")
        advance(MIN_GAP_MS.chat - 1)
        player.play("chat")
        player.play("click")
        expect(synths.chat).toHaveBeenCalledTimes(1)
        expect(synths.click).toHaveBeenCalledTimes(1)

        advance(1)
        player.play("chat")
        expect(synths.chat).toHaveBeenCalledTimes(2)
    })

    it("plays once a suspended context has woken up", async () => {
        const {player, synths, context} = setup({state: "suspended"})
        player.unlock()
        context.state = "suspended"

        player.play("click")
        expect(synths.click).not.toHaveBeenCalled()
        await vi.waitFor(() => expect(synths.click).toHaveBeenCalledTimes(1))
    })

    it("previews a sound whatever the settings say", () => {
        const {player, synths} = setup({audible: () => false})
        player.preview("refused")
        expect(synths.refused).toHaveBeenCalledTimes(1)
    })

    it("makes no context at a gesture while every sound is off", () => {
        let on = false
        const {player, synths, createContext} = setup({audible: () => on})
        player.unlock()
        expect(createContext).not.toHaveBeenCalled()

        on = true
        player.unlock()
        player.play("click")
        expect(createContext).toHaveBeenCalledTimes(1)
        expect(synths.click).toHaveBeenCalledTimes(1)
    })

    it("suspends the context once nothing has played for a while", () => {
        const {player, context} = setup()
        player.unlock()
        player.play("click")

        vi.advanceTimersByTime(SILENCE_MS - 1)
        expect(context.suspend).not.toHaveBeenCalled()

        vi.advanceTimersByTime(1)
        expect(context.suspend).toHaveBeenCalledTimes(1)
    })

    it("holds the context awake while the sounds keep coming", () => {
        const {player, context, advance} = setup()
        player.unlock()
        for (let i = 0; i < 5; i++) {
            vi.advanceTimersByTime(SILENCE_MS / 2)
            advance(SILENCE_MS / 2)
            player.play("click")
        }
        expect(context.suspend).not.toHaveBeenCalled()
    })

    it("wakes the suspended context for the next sound", async () => {
        const {player, synths, context} = setup()
        player.unlock()
        await vi.advanceTimersByTimeAsync(SILENCE_MS)
        expect(context.state).toBe("suspended")

        player.play("bonusSpawn")
        expect(context.resume).toHaveBeenCalled()
        await vi.waitFor(() => expect(synths.bonusSpawn).toHaveBeenCalledTimes(1))
    })

    it("wakes the suspended context at a gesture, and lets it rest again", async () => {
        const {player, context} = setup()
        player.unlock()
        await vi.advanceTimersByTimeAsync(SILENCE_MS)

        player.unlock()
        expect(context.resume).toHaveBeenCalledTimes(1)
        await vi.advanceTimersByTimeAsync(SILENCE_MS)
        expect(context.suspend).toHaveBeenCalledTimes(2)
    })

    it("waits for a suspend under way before it plays", async () => {
        const {player, synths, context} = setup()
        let suspended = () => {
        }
        context.suspend.mockImplementationOnce(() => new Promise<void>((done) => {
            suspended = () => {
                context.state = "suspended"
                done()
            }
        }))
        player.unlock()
        vi.advanceTimersByTime(SILENCE_MS)

        player.play("click")
        expect(synths.click).not.toHaveBeenCalled()
        suspended()
        await vi.waitFor(() => expect(synths.click).toHaveBeenCalledTimes(1))
        expect(context.state).toBe("running")
    })
})
