export const COMPACT = "(max-width: 768px)"

export const opensFolded = () => window.matchMedia?.(COMPACT).matches ?? false
