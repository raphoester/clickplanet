// The video page shares the game's origin: it must neither read nor write the player's storage, where the globe would
// find the ghosts' tiles and the settings.
function memoryStorage(): Storage {
    const kept = new Map<string, string>()
    return {
        get length() {
            return kept.size
        },
        clear: () => kept.clear(),
        getItem: (key) => kept.get(key) ?? null,
        key: (index) => [...kept.keys()][index] ?? null,
        removeItem: (key) => {
            kept.delete(key)
        },
        setItem: (key, value) => {
            kept.set(key, String(value))
        },
    }
}

Object.defineProperty(window, "localStorage", {value: memoryStorage(), configurable: true})
Object.defineProperty(window, "sessionStorage", {value: memoryStorage(), configurable: true})
