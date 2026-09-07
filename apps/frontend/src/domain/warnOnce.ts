const warned = new Set<string>()

/**
 * Logs a message the first time it is seen and never again.
 *
 * The things worth warning about here — an unknown country code, a missing
 * sprite region — arrive on every frame of a live update stream once they
 * arrive at all, so warning each time buries the console.
 */
export function warnOnce(message: string) {
    if (warned.has(message)) return
    warned.add(message)
    console.warn(message)
}

/** Test seam: lets a case assert on the first warning rather than the cache. */
export function resetWarnOnce() {
    warned.clear()
}
