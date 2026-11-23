// Mono 16-bit PCM WAV of samples in [-1, 1]: what ffmpeg reads, with no dependency.
export function wavOf(samples, rate) {
    const data = samples.length * 2
    const buffer = Buffer.alloc(44 + data)
    buffer.write("RIFF", 0)
    buffer.writeUInt32LE(36 + data, 4)
    buffer.write("WAVE", 8)
    buffer.write("fmt ", 12)
    buffer.writeUInt32LE(16, 16)
    buffer.writeUInt16LE(1, 20)
    buffer.writeUInt16LE(1, 22)
    buffer.writeUInt32LE(rate, 24)
    buffer.writeUInt32LE(rate * 2, 28)
    buffer.writeUInt16LE(2, 32)
    buffer.writeUInt16LE(16, 34)
    buffer.write("data", 36)
    buffer.writeUInt32LE(data, 40)
    for (let i = 0; i < samples.length; i++) {
        const sample = Math.max(-1, Math.min(1, samples[i]))
        buffer.writeInt16LE(Math.round(sample < 0 ? sample * 0x8000 : sample * 0x7fff), 44 + i * 2)
    }
    return buffer
}

// The sound without the silence Kokoro leaves before and after it, a margin kept so a soft first or last sound stays.
export function trimmed(samples, rate, threshold = 0.01, marginSeconds = 0.04) {
    let first = 0
    while (first < samples.length && Math.abs(samples[first]) < threshold) first++
    let last = samples.length - 1
    while (last > first && Math.abs(samples[last]) < threshold) last--
    if (first >= last) return samples
    const margin = Math.round(rate * marginSeconds)
    return samples.subarray(Math.max(0, first - margin), Math.min(samples.length, last + 1 + margin))
}

// The sentences one after the other, a pause between two.
export function joined(parts, rate, pauseSeconds) {
    const pause = Math.round(rate * pauseSeconds)
    const length = parts.reduce((sum, part) => sum + part.length, 0) + pause * Math.max(0, parts.length - 1)
    const samples = new Float32Array(length)
    let at = 0
    parts.forEach((part, i) => {
        samples.set(part, at)
        at += part.length + (i < parts.length - 1 ? pause : 0)
    })
    return samples
}
