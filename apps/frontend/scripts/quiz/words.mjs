const IGNORED = new Set(["the", "and"])

export function repeats(text, answer) {
    const asked = new Set(wordsOf(text))
    return wordsOf(answer).some((word) => asked.has(word))
}

export function names(text, name) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    return new RegExp(`(?<![\\p{L}\\p{N}])${escaped}(?![\\p{L}\\p{N}])`, "u").test(text)
}

function wordsOf(text) {
    return text
        .normalize("NFD")
        .replace(/\p{M}/gu, "")
        .toLowerCase()
        .split(/[^\p{L}\p{N}]+/u)
        .filter((word) => word.length >= 3 && !IGNORED.has(word))
}
