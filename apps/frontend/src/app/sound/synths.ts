import {IMPACT_DELAY} from "../../domain/blast.ts";
import {SoundName} from "../../domain/soundSettings.ts";
import {TITLE_REVEAL} from "../../domain/titleReveal.ts";

export type Synth = (ctx: AudioContext, at: number, options: SynthOptions) => void

export type SynthOptions = {
    volume: number
    onWater: boolean
}

type Tone = {
    type: OscillatorType
    from: number
    to?: number
    start: number
    length: number
    gain: number
}

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

const detune = (spread: number) => 1 + (Math.random() * 2 - 1) * spread

const click: Synth = (ctx, at, {volume}) => {
    const pitch = detune(0.08)
    tone(ctx, at, volume, {type: "sine", from: 720 * pitch, to: 260 * pitch, start: 0, length: 0.07, gain: 0.25})
}

const refused: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "triangle", from: 240, to: 220, start: 0, length: 0.09, gain: 0.35})
    tone(ctx, at, volume, {type: "triangle", from: 170, to: 150, start: 0.1, length: 0.16, gain: 0.35})
}

const bonusSpawn: Synth = (ctx, at, {volume}) => {
    ;[1047, 1319, 1568, 2093].forEach((from, i) => {
        tone(ctx, at, volume, {type: "sine", from, start: i * 0.06, length: 0.22, gain: 0.14})
    })
}

const bonusCaught: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "square", from: 988, start: 0, length: 0.08, gain: 0.07})
    tone(ctx, at, volume, {type: "square", from: 1319, start: 0.08, length: 0.45, gain: 0.07})
}

const spread: Synth = (ctx, at, {volume}) => {
    const pitch = detune(0.05)
    tone(ctx, at, volume, {type: "sine", from: 320 * pitch, to: 140 * pitch, start: 0, length: 0.12, gain: 0.3})
    ;[660, 784, 880, 1047, 1175, 1319].forEach((from, i) => {
        tone(ctx, at, volume, {type: "sine", from: from * pitch, to: from * pitch * 1.6, start: 0.03 + i * 0.035, length: 0.06, gain: 0.1})
    })
}

const enclose: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "triangle", from: 220, to: 880, start: 0, length: 0.5, gain: 0.12})
    ;[523, 659, 784, 1047].forEach((from) => {
        tone(ctx, at, volume, {type: "triangle", from, start: 0.5, length: 0.7, gain: 0.08})
    })
    for (let i = 0; i < 8; i++) {
        tone(ctx, at, volume, {type: "sine", from: 1568 + Math.random() * 1200, start: 0.52 + i * 0.055, length: 0.12, gain: 0.05})
    }
    tone(ctx, at, volume, {type: "sine", from: 262, start: 0.55, length: 0.9, gain: 0.2})
    tone(ctx, at, volume, {type: "sine", from: 196, start: 0.8, length: 1.1, gain: 0.18})
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

function filteredNoise(ctx: AudioContext, at: number, out: AudioNode, {start, length, gain, attack = 0.005, filter, from, to, q}: Noise) {
    const t = at + start
    const source = ctx.createBufferSource()
    source.buffer = noise(ctx)
    source.loop = true
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

type Rig = {
    bus: GainNode
    impact: number
    release: (lastsUntil: number) => void
}

function bombRig(ctx: AudioContext, at: number, volume: number, roomSize: number): Rig {
    const impact = at + IMPACT_DELAY

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
    wet.gain.value = roomSize

    const level = ctx.createGain()
    level.gain.value = volume

    bus.connect(drive).connect(limiter).connect(level).connect(ctx.destination)
    level.connect(room).connect(wet).connect(ctx.destination)

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

    const nodes: AudioNode[] = [whistleGain, wobbleDepth, bus, drive, limiter, level, room, wet]
    return {
        bus,
        impact,
        release: (lastsUntil) => window.setTimeout(
            () => nodes.forEach((node) => node.disconnect()),
            (lastsUntil - ctx.currentTime + 4.5) * 1000,
        ),
    }
}

function landBlast(ctx: AudioContext, {bus, impact, release}: Rig) {
    const tail = 4.5

    filteredNoise(ctx, impact, bus, {start: 0, length: 0.12, attack: 0.001, gain: 1.4, filter: "highpass", from: 1500})
    filteredNoise(ctx, impact, bus, {start: 0, length: 2.6, attack: 0.004, gain: 1.6, filter: "lowpass", from: 5000, to: 90, q: 0.9})

    tone(ctx, impact, 1, {type: "sine", from: 110, to: 26, start: 0, length: 1.6, gain: 1.8}, bus)
    tone(ctx, impact, 1, {type: "triangle", from: 165, to: 40, start: 0, length: 0.9, gain: 0.7}, bus)

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
    pulse.onended = () => {
        pulseDepth.disconnect()
        rumble.disconnect()
    }
    filteredNoise(ctx, impact, rumble, {start: 0.15, length: tail - 0.15, attack: 0.3, gain: 1.1, filter: "lowpass", from: 220, to: 50, q: 1.2})

    release(impact + tail)
}

function waterBlast(ctx: AudioContext, {bus, impact, release}: Rig) {
    const tail = 3

    tone(ctx, impact, 1, {type: "sine", from: 95, to: 32, start: 0, length: 1.1, gain: 1.6}, bus)
    filteredNoise(ctx, impact, bus, {start: 0, length: 1.3, attack: 0.01, gain: 1.4, filter: "lowpass", from: 900, to: 110, q: 0.8})

    filteredNoise(ctx, impact, bus, {start: 0.02, length: 1.5, attack: 0.07, gain: 1.3, filter: "bandpass", from: 900, to: 3200, q: 0.6})

    filteredNoise(ctx, impact, bus, {start: 0.35, length: tail - 0.35, attack: 0.5, gain: 0.55, filter: "highpass", from: 2800, to: 1600})

    for (let i = 0; i < 45; i++) {
        const when = 0.15 + (tail - 0.3) * Math.random() ** 1.8
        const from = 450 + Math.random() * 1100
        const fade = 1 - when / tail
        tone(ctx, impact, 1, {
            type: "sine",
            from,
            to: from * (2 + Math.random()),
            start: when,
            length: 0.025 + Math.random() * 0.035,
            gain: (0.08 + Math.random() * 0.25) * fade,
        }, bus)
    }

    release(impact + tail)
}

const bomb: Synth = (ctx, at, {volume, onWater}) => {
    if (onWater) waterBlast(ctx, bombRig(ctx, at, volume, 0.35))
    else landBlast(ctx, bombRig(ctx, at, volume, 0.55))
}

const quiz: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "triangle", from: 784, start: 0, length: 0.1, gain: 0.16})
    tone(ctx, at, volume, {type: "triangle", from: 1047, start: 0.09, length: 0.1, gain: 0.16})
    tone(ctx, at, volume, {type: "triangle", from: 1319, to: 1661, start: 0.18, length: 0.3, gain: 0.14})
}

const quizRight: Synth = (ctx, at, {volume}) => {
    ;[523, 659, 784].forEach((from) => {
        tone(ctx, at, volume, {type: "triangle", from, start: 0, length: 0.32, gain: 0.06})
    })
    tone(ctx, at, volume, {type: "sine", from: 1047, start: 0.1, length: 0.5, gain: 0.08})
}

const quizWrong: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "sine", from: 440, to: 415, start: 0, length: 0.14, gain: 0.18})
    tone(ctx, at, volume, {type: "sine", from: 349, to: 311, start: 0.13, length: 0.34, gain: 0.15})
}

const chat: Synth = (ctx, at, {volume}) => {
    tone(ctx, at, volume, {type: "sine", from: 880, start: 0, length: 0.08, gain: 0.12})
    tone(ctx, at, volume, {type: "sine", from: 1175, start: 0.07, length: 0.14, gain: 0.12})
}

const ROLL = 18

const title: Synth = (ctx, at, {volume}) => {
    const hit = TITLE_REVEAL.impact

    const limiter = ctx.createDynamicsCompressor()
    limiter.threshold.value = -10
    limiter.ratio.value = 6
    const level = ctx.createGain()
    level.gain.value = volume
    const room = ctx.createConvolver()
    room.buffer = reverb(ctx)
    const wet = ctx.createGain()
    wet.gain.value = 0.3
    limiter.connect(level).connect(ctx.destination)
    level.connect(room).connect(wet).connect(ctx.destination)

    filteredNoise(ctx, at, limiter, {start: 0, length: hit, attack: hit * 0.95, gain: 0.12, filter: "bandpass", from: 400, to: 5000, q: 1.2})
    for (let i = 0; i < ROLL; i++) {
        const start = 0.2 + (hit - 0.24) * (1 - (1 - i / ROLL) ** 1.6)
        filteredNoise(ctx, at, limiter, {start, length: 0.05, attack: 0.002, gain: 0.05 + 0.25 * i / ROLL, filter: "bandpass", from: 2000, q: 0.8})
    }

    tone(ctx, at, 1, {type: "sine", from: 150, to: 40, start: hit, length: 0.9, gain: 0.7}, limiter)
    filteredNoise(ctx, at, limiter, {start: hit, length: 1.8, attack: 0.002, gain: 0.3, filter: "highpass", from: 2500})
    ;[262, 523, 659, 784].forEach((from) => {
        tone(ctx, at, 1, {type: "square", from, start: hit, length: 0.14, gain: 0.05}, limiter)
    })
    ;[523, 659, 784, 1047].forEach((from) => {
        tone(ctx, at, 1, {type: "triangle", from, start: hit + 0.16, length: 1.9, gain: 0.09}, limiter)
        tone(ctx, at, 1, {type: "square", from, start: hit + 0.16, length: 1.2, gain: 0.02}, limiter)
    })
    ;[1568, 2093, 2637, 3136].forEach((from, i) => {
        tone(ctx, at, 1, {type: "sine", from, start: hit + 0.2 + i * 0.06, length: 0.3, gain: 0.05}, limiter)
    })

    window.setTimeout(
        () => [limiter, level, room, wet].forEach((node) => node.disconnect()),
        (at + hit - ctx.currentTime + 6) * 1000,
    )
}

// A gate slamming shut, then the shields ringing in as the wave runs over the land.
const fortify: Synth = (ctx, at, {volume}) => {
    const out = ctx.createGain()
    out.gain.value = 1
    out.connect(ctx.destination)
    filteredNoise(ctx, at, out, {start: 0, length: 0.18, gain: 0.35 * volume, filter: "lowpass", from: 900, to: 120})
    tone(ctx, at, volume, {type: "sine", from: 110, to: 55, start: 0, length: 0.35, gain: 0.45}, out)
    tone(ctx, at, volume, {type: "triangle", from: 196, start: 0.02, length: 0.5, gain: 0.12}, out)
    ;[784, 988, 1175, 1568, 1976].forEach((from, i) => {
        tone(ctx, at, volume, {type: "sine", from, start: 0.18 + i * 0.16, length: 0.9, gain: 0.07}, out)
        tone(ctx, at, volume, {type: "sine", from: from * 2.76, start: 0.18 + i * 0.16, length: 0.35, gain: 0.025}, out)
    })
    window.setTimeout(() => out.disconnect(), (at - ctx.currentTime + 2.5) * 1000)
}

export const SYNTHS: Record<SoundName, Synth> = {
    click, refused, bonusSpawn, bonusCaught, spread, enclose, bomb, chat, quiz, quizRight, quizWrong, title, fortify,
}
