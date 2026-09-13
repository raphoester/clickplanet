import {describe, expect, it, vi} from "vitest"
import {createSoundPlayer, MIN_GAP_MS} from "./soundPlayer.ts"
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
        close: vi.fn(async () => {
        }),
    }
    const synths = Object.fromEntries(SOUNDS.map((name) => [name, vi.fn<Synth>()])) as Record<SoundName, ReturnType<typeof vi.fn<Synth>>>
    const player = createSoundPlayer({
        isAudible: overrides.audible ?? (() => true),
        createContext: () => context as unknown as AudioContext,
        synths,
        now: () => clock,
        isHidden: () => overrides.hidden ?? false,
    })
    return {player, synths, context, advance: (ms: number) => clock += ms}
}

describe("createSoundPlayer", () => {
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
})
