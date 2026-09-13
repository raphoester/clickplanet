import {IMPACT_DELAY} from "../../domain/blast.ts";
import {SoundName} from "../../domain/soundSettings.ts";

/**
 * Every sound in the game, built from oscillators and noise at the moment it
 * plays — there is no audio file anywhere. A synth schedules its nodes from
 * `at` on the context's own clock; they disconnect themselves once stopped.
 */
export type Synth = (ctx: AudioContext, at: number, volume: number) => void

type Tone = {
    type: OscillatorType
    from: number
    to?: number
    start: number
    length: number
    gain: number
}

/** One oscillator with a fast attack and an exponential fade. */
function tone(ctx: AudioContext, at: number, volume: number, {type, from, to, start, length, gain}: Tone, out: AudioNode = ctx.destination) {
    const t = at + start
    const osc = ctx.createOscillator()
    const envelope = ctx.createGain()

    osc.type = type
    osc.frequency.setValueAtTime(from, t)
    if (to !== undefined) osc.frequency.exponentialRampToValueAtTime(to, t + length)

    // Exponential ramps cannot reach zero, and a ramp from silence clicks.
    envelope.gain.setValueAtTime(0.0001, t)
    envelope.gain.exponentialRampToValueAtTime(gain * volume, t + 0.005)
    envelope.gain.exponentialRampToValueAtTime(0.0001, t + length)

    osc.connect(envelope).connect(out)
    osc.start(t)
    osc.stop(t + length + 0.02)
    osc.onended = () => envelope.disconnect()
}

const noiseBuffers = new WeakMap<BaseAudioContext, AudioBuffer>()

/** Two seconds of white noise, made once per context. */
function noise(ctx: AudioContext): AudioBuffer {
    let buffer = noiseBuffers.get(ctx)
    if (!buffer) {
        buffer = ctx.createBuffer(1, ctx.sampleRate * 2, ctx.sampleRate)
        const data = buffer.getChannelData(0)
        for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1
        noiseBuffers.set(ctx, buffer)
    }
    return buffer
}

/** Up to ±`spread` of a frequency, so a sound heard fifty times is never quite the same twice. */
const detune = (spread: number) => 1 + (Math.random() * 2 - 1) * spread

const click: Synth = (ctx, at, volume) => {
    const pitch = detune(0.08)
    tone(ctx, at, volume, {type: "sine", from: 720 * pitch, to: 260 * pitch, start: 0, length: 0.07, gain: 0.25})
}

// Two falling notes, low and dull: "nope".
const refused: Synth = (ctx, at, volume) => {
    tone(ctx, at, volume, {type: "triangle", from: 240, to: 220, start: 0, length: 0.09, gain: 0.35})
    tone(ctx, at, volume, {type: "triangle", from: 170, to: 150, start: 0.1, length: 0.16, gain: 0.35})
}

// A rising sparkle, C major up an octave: something appeared.
const bonusSpawn: Synth = (ctx, at, volume) => {
    ;[1047, 1319, 1568, 2093].forEach((from, i) => {
        tone(ctx, at, volume, {type: "sine", from, start: i * 0.06, length: 0.22, gain: 0.14})
    })
}

// The coin: a short note and a long one a fourth above.
const bonusCaught: Synth = (ctx, at, volume) => {
    tone(ctx, at, volume, {type: "square", from: 988, start: 0, length: 0.08, gain: 0.07})
    tone(ctx, at, volume, {type: "square", from: 1319, start: 0.08, length: 0.45, gain: 0.07})
}

type Noise = {
    start: number
    length: number
    gain: number
    attack?: number
    filter: BiquadFilterType
    from: number
    to?: number
    q?: number
}

/** A stretch of filtered noise with a fast attack and an exponential fade. */
function filteredNoise(ctx: AudioContext, at: number, out: AudioNode, {start, length, gain, attack = 0.005, filter, from, to, q}: Noise) {
    const t = at + start
    const source = ctx.createBufferSource()
    source.buffer = noise(ctx)
    source.loop = true
    // A random start into the buffer, so two blasts are not the same noise.
    const offset = Math.random() * 1.5

    const shape = ctx.createBiquadFilter()
    shape.type = filter
    shape.frequency.setValueAtTime(from, t)
    if (to !== undefined) shape.frequency.exponentialRampToValueAtTime(to, t + length)
    if (q !== undefined) shape.Q.value = q

    const envelope = ctx.createGain()
    envelope.gain.setValueAtTime(0.0001, t)
    envelope.gain.exponentialRampToValueAtTime(gain, t + attack)
    envelope.gain.exponentialRampToValueAtTime(0.0001, t + length)

    source.connect(shape).connect(envelope).connect(out)
    source.start(t, offset)
    source.stop(t + length + 0.05)
    source.onended = () => {
        shape.disconnect()
        envelope.disconnect()
    }
}

const reverbs = new WeakMap<BaseAudioContext, AudioBuffer>()

/** A big, dark room: four seconds of stereo noise dying away, made once per context. */
function reverb(ctx: AudioContext): AudioBuffer {
    let buffer = reverbs.get(ctx)
    if (!buffer) {
        const length = ctx.sampleRate * 4
        buffer = ctx.createBuffer(2, length, ctx.sampleRate)
        for (let channel = 0; channel < 2; channel++) {
            const data = buffer.getChannelData(channel)
            for (let i = 0; i < length; i++) data[i] = (Math.random() * 2 - 1) * (1 - i / length) ** 3
        }
        reverbs.set(ctx, buffer)
    }
    return buffer
}

const saturation = new Float32Array(1024).map((_, i) => {
    const x = (i / 1023) * 2 - 1
    return Math.tanh(3 * x)
})

// The bomb, the one sound meant to be felt. The drop is heard when it is
// broadcast, so the fall takes `IMPACT_DELAY` and the blast lands on the frame
// the tiles go, exactly as the drawing does.
//
// Everything runs through a soft clipper (grit, and the harmonics that let a
// laptop speaker suggest a sub it cannot play), a compressor (loud without
// clipping), and a long reverb (the size of the thing).
const bomb: Synth = (ctx, at, volume) => {
    const impact = at + IMPACT_DELAY
    const tail = 4.5

    // The layers below add up well past 1, which the clipper would square off.
    const bus = ctx.createGain()
    bus.gain.value = 0.4

    const drive = ctx.createWaveShaper()
    drive.curve = saturation
    drive.oversample = "2x"

    const limiter = ctx.createDynamicsCompressor()
    limiter.threshold.value = -14
    limiter.knee.value = 6
    limiter.ratio.value = 8
    limiter.attack.value = 0.002
    limiter.release.value = 0.4

    const room = ctx.createConvolver()
    room.buffer = reverb(ctx)
    const wet = ctx.createGain()
    wet.gain.value = 0.55

    // Volume after the dynamics, so a distant bomb is quieter, not cleaner.
    const level = ctx.createGain()
    level.gain.value = volume

    bus.connect(drive).connect(limiter).connect(level).connect(ctx.destination)
    level.connect(room).connect(wet).connect(ctx.destination)

    // The fall: a falling whistle with a wobble, over rushing air.
    const whistle = ctx.createOscillator()
    const whistleGain = ctx.createGain()
    const wobble = ctx.createOscillator()
    const wobbleDepth = ctx.createGain()
    whistle.type = "sine"
    whistle.frequency.setValueAtTime(1800, at)
    whistle.frequency.exponentialRampToValueAtTime(420, impact)
    wobble.frequency.value = 9
    wobbleDepth.gain.value = 18
    wobble.connect(wobbleDepth).connect(whistle.frequency)
    whistleGain.gain.setValueAtTime(0.0001, at)
    whistleGain.gain.exponentialRampToValueAtTime(0.12, impact - 0.1)
    whistleGain.gain.exponentialRampToValueAtTime(0.0001, impact)
    whistle.connect(whistleGain).connect(bus)
    whistle.start(at)
    wobble.start(at)
    whistle.stop(impact + 0.02)
    wobble.stop(impact + 0.02)
    filteredNoise(ctx, at, bus, {start: 0, length: IMPACT_DELAY, attack: IMPACT_DELAY * 0.9, gain: 0.08, filter: "bandpass", from: 600, to: 2400, q: 1.5})

    // The crack: a split second of bright noise, the edge of the blast.
    filteredNoise(ctx, impact, bus, {start: 0, length: 0.12, attack: 0.001, gain: 1.4, filter: "highpass", from: 1500})

    // The body: wide noise closing down from bright to dark.
    filteredNoise(ctx, impact, bus, {start: 0, length: 2.6, attack: 0.004, gain: 1.6, filter: "lowpass", from: 5000, to: 90, q: 0.9})

    // The sub drop, and a fifth above it so it survives a small speaker.
    tone(ctx, impact, 1, {type: "sine", from: 110, to: 26, start: 0, length: 1.6, gain: 1.8}, bus)
    tone(ctx, impact, 1, {type: "triangle", from: 165, to: 40, start: 0, length: 0.9, gain: 0.7}, bus)

    // The rumble that rolls on after, pulsing like debris coming down.
    const rumble = ctx.createGain()
    const pulse = ctx.createOscillator()
    const pulseDepth = ctx.createGain()
    pulse.frequency.setValueAtTime(7, impact)
    pulse.frequency.linearRampToValueAtTime(2, impact + tail)
    pulseDepth.gain.value = 0.4
    rumble.gain.value = 0.6
    pulse.connect(pulseDepth).connect(rumble.gain)
    rumble.connect(bus)
    pulse.start(impact)
    pulse.stop(impact + tail)
    filteredNoise(ctx, impact, rumble, {start: 0.15, length: tail - 0.15, attack: 0.3, gain: 1.1, filter: "lowpass", from: 220, to: 50, q: 1.2})

    // One node graph per blast, torn down once the reverb has rung out.
    pulse.onended = () => {
        pulse.disconnect()
        pulseDepth.disconnect()
        rumble.disconnect()
        whistleGain.disconnect()
        wobbleDepth.disconnect()
        window.setTimeout(() => {
            bus.disconnect()
            drive.disconnect()
            limiter.disconnect()
            level.disconnect()
            room.disconnect()
            wet.disconnect()
        }, 4000)
    }
}

// Two soft notes going up: someone said something.
const chat: Synth = (ctx, at, volume) => {
    tone(ctx, at, volume, {type: "sine", from: 880, start: 0, length: 0.08, gain: 0.12})
    tone(ctx, at, volume, {type: "sine", from: 1175, start: 0.07, length: 0.14, gain: 0.12})
}

export const SYNTHS: Record<SoundName, Synth> = {click, refused, bonusSpawn, bonusCaught, bomb, chat}
