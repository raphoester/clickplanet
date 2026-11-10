import {MapHistory} from "../mapHistory.ts"
import {ranked, tally} from "./changes.ts"
import {PlaceFacts, placeKeysOf, placeOf} from "./findings.ts"
import {RegionOf} from "./story.ts"

// A flag holding over half of a place, from since to until (undefined while it holds). pushedAt: the last moment it held
// under a tenth of the place before since, when its conquest began; undefined when it held a tenth from the start.
// tookFrom: the flag holding the most of the place after it then. lostTo: the flag holding the most after it when it fell.
// share: its share at until, or at the end while it holds.
export type Reign = {
    flag: string
    pushedAt: number | undefined
    since: number
    until: number | undefined
    peak: number
    share: number
    tookFrom: string | undefined
    tookFromShare: number
    lostTo: string | undefined
}

// A place over a history, and each flag that held over half of it, oldest first; its holder and share at the end.
export type SagaFacts = Pick<PlaceFacts, "kind" | "place" | "code" | "continent"> & {
    tiles: number
    reigns: Reign[]
    holder: string | undefined
    share: number
    since: number
    until: number
}

const HELD = 0.5

const PUSHED = 0.1

// A dip under half this short does not end a reign, and a reign this short that ended is a raid, not a reign.
const DIP_MS = 3 * 3_600_000

// The flags each frame keeps beside every flag over a tenth: enough to name who took it and who it fell to.
const KEPT = 5

type Seen = {at: number, top: [string, number][], above: ReadonlyMap<string, number>}

// Each place of the map over a history, from the map at each of its frames. A share is of all the place's tiles, empty
// ones too, as a place's is. A place no flag ever held over half of is left out, and so is a country that is its whole
// continent: Antarctica is told once.
export function sagasOf(
    history: MapHistory,
    tiles: number,
    groundOf: (tile: number) => string | undefined,
    regionOf: RegionOf,
): SagaFacts[] {
    const frames = history.frames
    if (frames.length === 0) return []
    const sizes = tally(Array.from({length: tiles}, (_, i) => groundOf(i + 1)).flatMap((ground) => ground ?? []))
    const names: string[] = []
    const indexes = new Map<string, number>()
    const keysOf = new Map<string | undefined, number[]>()
    const tileKeys: number[][] = [[]]
    for (let tile = 1; tile <= tiles; tile++) {
        const ground = groundOf(tile)
        let keys = keysOf.get(ground)
        if (keys === undefined) {
            keys = placeKeysOf(ground, sizes, regionOf).map((key) => {
                let index = indexes.get(key)
                if (index === undefined) {
                    index = names.length
                    names.push(key)
                    indexes.set(key, index)
                }
                return index
            })
            keysOf.set(ground, keys)
        }
        tileKeys.push(keys)
    }
    const grounds = new Array<number>(names.length).fill(0)
    for (let tile = 1; tile <= tiles; tile++) for (const key of tileKeys[tile]) grounds[key]++

    const owners = new Array<string>(tiles + 1).fill("")
    const counts = names.map(() => new Map<string, number>())
    const seen: Seen[][] = names.map(() => [])
    for (const frame of frames) {
        frame.tiles.forEach((tile, i) => {
            if (tile < 1 || tile > tiles) return
            const was = owners[tile]
            const now = frame.owners[i]
            if (was === now) return
            owners[tile] = now
            for (const key of tileKeys[tile]) {
                const held = counts[key]
                if (was !== "") held.set(was, (held.get(was) ?? 0) - 1)
                if (now !== "") held.set(now, (held.get(now) ?? 0) + 1)
            }
        })
        counts.forEach((held, key) => {
            const least = PUSHED * grounds[key]
            const top = ranked(held).filter(([, count]) => count > 0)
            seen[key].push({at: frame.at, top: top.slice(0, KEPT), above: new Map(top.filter(([, count]) => count >= least))})
        })
    }

    const sagas = names.flatMap((name, key): SagaFacts[] => {
        const ground = grounds[key]
        const reigns = reignsOf(seen[key], ground)
        if (reigns.length === 0) return []
        const [holder, held] = seen[key][seen[key].length - 1].top[0] ?? [undefined, 0]
        return [{
            ...placeOf(name, regionOf), tiles: ground, reigns, holder, share: held / Math.max(1, ground),
            since: frames[0].at, until: frames[frames.length - 1].at,
        }]
    })
    const wholeContinent = (saga: SagaFacts) => saga.kind === "country"
        && sagas.some(({kind, continent, tiles: size}) => kind === "continent" && continent === saga.continent && size === saga.tiles)
    return sagas.filter((saga) => !wholeContinent(saga)).sort((a, b) => b.tiles - a.tiles)
}

function reignsOf(seen: readonly Seen[], ground: number): Reign[] {
    const countOf = (frame: Seen, flag: string) => frame.above.get(flag) ?? frame.top.find(([code]) => code === flag)?.[1] ?? 0
    const shareOf = (frame: Seen, flag: string) => countOf(frame, flag) / Math.max(1, ground)
    const nextAfter = (frame: Seen, flag: string) => frame.top.find(([code]) => code !== flag)
    const holderAt = seen.map(({top}) => top[0] !== undefined && top[0][1] > HELD * ground ? top[0][0] : undefined)

    const reigns: Reign[] = []
    let i = 0
    while (i < seen.length) {
        const flag = holderAt[i]
        if (flag === undefined) {
            i++
            continue
        }
        const start = i
        let peak = 0
        let j = i
        while (j < seen.length) {
            if (holderAt[j] === flag) {
                peak = Math.max(peak, shareOf(seen[j], flag))
                j++
                continue
            }
            let k = j
            while (k < seen.length && holderAt[k] === undefined && seen[k].at - seen[j].at < DIP_MS) k++
            if (k < seen.length && holderAt[k] === flag && seen[k].at - seen[j].at < DIP_MS) {
                j = k
                continue
            }
            break
        }

        let pushed: number | undefined
        for (let p = start - 1; p >= 0 && pushed === undefined; p--) if (!seen[p].above.has(flag)) pushed = p
        const before = nextAfter(seen[pushed ?? 0], flag)
        const fell = j < seen.length ? seen[j] : undefined
        const lost = fell === undefined ? undefined : nextAfter(fell, flag)
        reigns.push({
            flag,
            pushedAt: pushed === undefined ? undefined : seen[pushed].at,
            since: seen[start].at,
            until: fell?.at,
            peak,
            share: shareOf(fell ?? seen[seen.length - 1], flag),
            tookFrom: before?.[0],
            tookFromShare: before === undefined ? 0 : before[1] / Math.max(1, ground),
            lostTo: lost?.[0],
        })
        i = j
    }
    return reigns.filter(({since, until}) => until === undefined || until - since >= DIP_MS)
}

// The tiles a country's saga is framed on: its landmasses of a quarter of its biggest one's size or more, so French
// Guiana does not pull the camera off France, nor Alaska off the USA. A continent is framed whole.
export function framedTilesOf(placeTiles: readonly number[], landmassOf: (tile: number) => number): number[] {
    const sizes = tally(placeTiles.map(landmassOf))
    const biggest = Math.max(0, ...sizes.values())
    return placeTiles.filter((tile) => (sizes.get(landmassOf(tile)) ?? 0) * 4 >= biggest)
}
