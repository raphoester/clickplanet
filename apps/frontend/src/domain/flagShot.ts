// The capital and the tiles around it, ring by ring: the ground a new flag is shown on before it is in the game.
export function groundAround(capital: number, rings: number, neighboursOf: (tile: number) => readonly number[]): number[] {
    const ground = [capital]
    const seen = new Set(ground)
    let ring = ground
    for (let step = 0; step < rings; step++) {
        const next: number[] = []
        for (const tile of ring) {
            for (const neighbour of neighboursOf(tile)) {
                if (seen.has(neighbour)) continue
                seen.add(neighbour)
                next.push(neighbour)
            }
        }
        ground.push(...next)
        ring = next
    }
    return ground
}

export type Rect = {x: number, y: number, width: number, height: number}

// A rectangle of a frame's size centred on a point, moved inside the frame when the point is near its edge.
export function cropAround(frame: {width: number, height: number}, at: {x: number, y: number}, size: {width: number, height: number}): Rect {
    const width = Math.min(Math.round(size.width), frame.width)
    const height = Math.min(Math.round(size.height), frame.height)
    const x = Math.min(Math.max(Math.round(at.x - width / 2), 0), frame.width - width)
    const y = Math.min(Math.max(Math.round(at.y - height / 2), 0), frame.height - height)
    return {x, y, width, height}
}
