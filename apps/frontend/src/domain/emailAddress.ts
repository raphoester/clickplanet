export function looksLikeAnAddress(address: string): boolean {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address.trim())
}
