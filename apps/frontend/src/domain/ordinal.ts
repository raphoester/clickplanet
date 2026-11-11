// 1st, 2nd, 3rd, 4th, 11th, 21st.
export function ordinalOf(rank: number): string {
    const teen = rank % 100 >= 11 && rank % 100 <= 13
    const suffix = teen ? "th" : ["th", "st", "nd", "rd"][rank % 10] ?? "th"
    return `${rank}${suffix}`
}
