/**
 * @param {string} csv RFC 4180, as Google Sheets exports it
 * @returns {string[]} the first cell of every row
 */
export function firstColumn(csv) {
    const cells = []
    let cell = ""
    let column = 0
    let quoted = false
    for (let i = 0; i < csv.length; i++) {
        const c = csv[i]
        if (quoted) {
            if (c === "\"" && csv[i + 1] === "\"") {
                cell += c
                i++
            } else if (c === "\"") {
                quoted = false
            } else {
                cell += c
            }
        } else if (c === "\"") {
            quoted = true
        } else if (c === "," || c === "\n" || c === "\r") {
            if (column === 0) cells.push(cell)
            cell = ""
            column = c === "," ? column + 1 : 0
            if (c === "\r" && csv[i + 1] === "\n") i++
        } else {
            cell += c
        }
    }
    if (column === 0 && cell !== "") cells.push(cell)
    return cells
}

/**
 * @param {string[]} cells
 * @returns {string[]} one entry per cell, its spaces made single, without blanks or repeats
 */
export function wordsOf(cells) {
    return [...new Set(cells.map((cell) => cell.replace(/\s+/gu, " ").trim()).filter((word) => word !== ""))]
}
