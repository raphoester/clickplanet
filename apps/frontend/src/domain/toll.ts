export type TollStep = {
    share: number
    slowdown: number
}

export type TollRow = {
    share: number
    slowdown: number
    here: boolean
}

export function slowdownAt(steps: readonly TollStep[], share: number): number {
    let slowdown = 1
    for (const step of steps) {
        if (share < step.share) break
        slowdown = step.slowdown
    }
    return slowdown
}

export function tollRows(steps: readonly TollStep[], share: number): TollRow[] {
    const rows = [{share: 0, slowdown: 1}, ...steps]
    let here = 0
    rows.forEach((row, index) => {
        if (share >= row.share) here = index
    })
    return rows.map((row, index) => ({...row, here: index === here}))
}
