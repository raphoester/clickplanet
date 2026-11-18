export type Cell = {x: number, y: number, width: number, height: number}

// Every cell sits on a multiple of its own size, so a mip level never mixes two cells.
export const ROOT = {width: 2048, height: 1024} as const

export const LEVELS = 9

// Some players' MAX_TEXTURE_SIZE is 4096.
const MOST_ROOTS = 4

export function cellSize(level: number): {width: number, height: number} {
    const halvings = Math.floor(level / 2)
    return level % 2 === 0
        ? {width: ROOT.width >> halvings, height: ROOT.height >> halvings}
        : {width: ROOT.height >> halvings, height: ROOT.height >> halvings}
}

export function levelOf({width, height}: {width: number, height: number}): number {
    return width > height ? 2 * Math.log2(ROOT.width / width) : 2 * Math.log2(ROOT.height / width) + 1
}

export function unitsOf(level: number): number {
    return 2 ** (LEVELS - 1 - level)
}

// Its last column and row repeated out to the cell's edge, so a coarse mip level finds the flag's own colours.
export function marginOf({width, height}: {width: number, height: number}): number {
    return Math.max(2, Math.min(width, height) / 32)
}

export function flagIn(cell: Cell, aspect: number): Cell {
    const margin = marginOf(cell)
    const roomWidth = cell.width - 2 * margin
    const roomHeight = cell.height - 2 * margin
    const width = Math.min(roomWidth, Math.round(roomHeight * aspect))
    const height = Math.min(roomHeight, Math.round(roomWidth / aspect))
    return {
        x: cell.x + Math.floor((cell.width - width) / 2),
        y: cell.y + Math.floor((cell.height - height) / 2),
        width,
        height,
    }
}

export function flagWidthAt(level: number, aspect: number): number {
    return flagIn({x: 0, y: 0, ...cellSize(level)}, aspect).width
}

export function levelFor(need: number, aspect: number, largest: number): number {
    for (let level = LEVELS - 1; level > largest; level--) {
        if (flagWidthAt(level, aspect) >= need) return level
    }
    return largest
}

export type SlotBudget = {
    roots: number
    largest: number
}

export function slotBudgetOf(width: number, height: number): SlotBudget {
    const roots = Math.min(MOST_ROOTS, Math.max(1, Math.round((width * height) / (ROOT.width * ROOT.height))))
    const widest = 2 ** Math.floor(Math.log2(Math.max(width, height, 1)))
    let largest = 0
    while (largest < LEVELS - 1 && cellSize(largest).width > widest) largest++
    return {roots, largest}
}

export function textureSizeOf({roots}: SlotBudget): {width: number, height: number} {
    return {width: ROOT.width, height: ROOT.height * roots}
}

export class CellPacking {
    private readonly free: Cell[][] = Array.from({length: LEVELS}, () => [])

    constructor(roots: number) {
        for (let root = 0; root < roots; root++) this.free[0].push({x: 0, y: root * ROOT.height, ...ROOT})
    }

    place(level: number): Cell | undefined {
        let from = level
        while (from >= 0 && this.free[from].length === 0) from--
        if (from < 0) return undefined
        let cell = this.take(from)
        for (let at = from; at < level; at++) {
            const [kept, spare] = halvesOf(cell)
            this.free[at + 1].push(spare)
            cell = kept
        }
        return cell
    }

    release(cell: Cell) {
        let level = levelOf(cell)
        let joined = cell
        while (level > 0) {
            const buddy = buddyOf(joined)
            const at = this.free[level].findIndex(({x, y}) => x === buddy.x && y === buddy.y)
            if (at < 0) break
            this.free[level].splice(at, 1)
            joined = {x: Math.min(joined.x, buddy.x), y: Math.min(joined.y, buddy.y), ...cellSize(level - 1)}
            level--
        }
        this.free[level].push(joined)
    }

    private take(level: number): Cell {
        const cells = this.free[level]
        let best = 0
        for (let i = 1; i < cells.length; i++) {
            if (cells[i].y < cells[best].y || (cells[i].y === cells[best].y && cells[i].x < cells[best].x)) best = i
        }
        return cells.splice(best, 1)[0]
    }
}

function halvesOf({x, y, width, height}: Cell): [Cell, Cell] {
    return width > height
        ? [{x, y, width: width / 2, height}, {x: x + width / 2, y, width: width / 2, height}]
        : [{x, y, width, height: height / 2}, {x, y: y + height / 2, width, height: height / 2}]
}

function buddyOf({x, y, width, height}: Cell): Cell {
    return width === height
        ? {x: x ^ width, y, width, height}
        : {x, y: y ^ height, width, height}
}

export function overlap(a: Cell, b: Cell): boolean {
    return a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height
}
