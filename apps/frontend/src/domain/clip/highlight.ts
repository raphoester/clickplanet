const SIZE = 4096
const HOP = 2048
const LOW_HZ = 250
const HIGH_HZ = 4000
const MOMENT_SECONDS = 8
const NOISY = 0.15
const NOISIER = 2
const NOISE = 0.25
const NOISY_SHARE = 0.1
const ONSET_DB = 3
const PLAYING_DB = 3
const PAUSE_SECONDS = 0.7
const PAUSE_DB = 9
const PHRASE_DB = 3
const MOST_HIGHLIGHTS = 3

export type Sound = {at: number, loud: number, flat: number}

// The moments a clip can start its anthem on, the best first: a note struck with the band playing its tune for the
// next seconds, loudest first, a phrase starting after a pause counting as louder. A drum roll is no tune: it is
// noisier than the band, which some recordings drown in cymbals. The first
// such note comes last, so a long clip on a short anthem still starts on one.
export function highlightsOf(sounds: readonly Sound[]): number[] {
    if (sounds.length < 2) return []
    const hop = sounds[1].at - sounds[0].at
    const loud = sounds.map((sound) => sound.loud).sort((a, b) => a - b)
    const middle = loud[Math.floor(loud.length / 2)]
    const flat = sounds.map((sound) => sound.flat).sort((a, b) => a - b)
    const noisy = Math.min(NOISE, Math.max(NOISY, NOISIER * flat[Math.floor(flat.length / 2)]))
    const pause = Math.max(1, Math.round(PAUSE_SECONDS / hop))
    const moment = Math.round(MOMENT_SECONDS / hop)

    const notes: {at: number, score: number}[] = []
    for (let i = 1; i + moment <= sounds.length; i++) {
        if (sounds[i].loud < middle - PLAYING_DB || sounds[i].loud - sounds[i - 1].loud < ONSET_DB) continue
        if (sounds[i].flat > noisy) continue
        const next = sounds.slice(i, i + moment)
        if (next.filter((sound) => sound.flat > noisy).length > next.length * NOISY_SHARE) continue
        const paused = Math.min(...sounds.slice(Math.max(0, i - pause), i).map((sound) => sound.loud)) <= middle - PAUSE_DB
        const score = next.reduce((sum, sound) => sum + sound.loud, 0) / next.length + (paused ? PHRASE_DB : 0)
        notes.push({at: sounds[i].at, score})
    }
    if (notes.length === 0) return []

    const apart = (chosen: readonly number[], at: number) => chosen.every((other) => Math.abs(other - at) >= MOMENT_SECONDS)
    const best = [...notes].sort((a, b) => b.score - a.score)
        .reduce<number[]>((chosen, note) => chosen.length < MOST_HIGHLIGHTS && apart(chosen, note.at) ? [...chosen, note.at] : chosen, [])
    return apart(best, notes[0].at) ? [...best, notes[0].at] : best
}

export function soundsOf(samples: Float32Array, rate: number): Sound[] {
    const window = Float32Array.from({length: SIZE}, (_, i) => 0.5 - 0.5 * Math.cos(2 * Math.PI * i / SIZE))
    const low = Math.ceil(LOW_HZ * SIZE / rate)
    const high = Math.floor(HIGH_HZ * SIZE / rate)
    const cos = Float64Array.from({length: SIZE / 2}, (_, k) => Math.cos(-2 * Math.PI * k / SIZE))
    const sin = Float64Array.from({length: SIZE / 2}, (_, k) => Math.sin(-2 * Math.PI * k / SIZE))
    const re = new Float64Array(SIZE)
    const im = new Float64Array(SIZE)
    const sounds: Sound[] = []
    for (let start = 0; start + SIZE <= samples.length; start += HOP) {
        for (let i = 0; i < SIZE; i++) re[i] = samples[start + i] * window[i]
        im.fill(0)
        fft(re, im, cos, sin)
        let power = 0
        let logs = 0
        for (let bin = low; bin <= high; bin++) {
            const p = re[bin] * re[bin] + im[bin] * im[bin] + 1e-12
            power += p
            logs += Math.log(p)
        }
        const bins = high - low + 1
        sounds.push({
            at: (start + SIZE / 2) / rate,
            loud: 10 * Math.log10(power / bins),
            flat: Math.exp(logs / bins) / (power / bins),
        })
    }
    return sounds
}

function fft(re: Float64Array, im: Float64Array, cos: Float64Array, sin: Float64Array): void {
    const n = re.length
    for (let i = 1, j = 0; i < n; i++) {
        let bit = n >> 1
        for (; j & bit; bit >>= 1) j ^= bit
        j ^= bit
        if (i < j) {
            [re[i], re[j]] = [re[j], re[i]];
            [im[i], im[j]] = [im[j], im[i]]
        }
    }
    for (let length = 2; length <= n; length <<= 1) {
        const step = n / length
        for (let i = 0; i < n; i += length) {
            for (let k = 0; k < length / 2; k++) {
                const c = cos[k * step]
                const s = sin[k * step]
                const a = i + k
                const b = a + length / 2
                const tr = re[b] * c - im[b] * s
                const ti = re[b] * s + im[b] * c
                re[b] = re[a] - tr
                im[b] = im[a] - ti
                re[a] += tr
                im[a] += ti
            }
        }
    }
}
