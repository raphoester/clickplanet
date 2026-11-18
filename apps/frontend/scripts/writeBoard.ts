import {createHash} from "node:crypto"
import {copyFileSync, existsSync, mkdirSync, readdirSync, rmSync, unlinkSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"

import type {Board, BoardAsset, BoardFiles, PageFiles} from "../src/domain/board.ts"
import {blobNamed, boardDir, boardsDir, readManifest, staticDir} from "./map/board.mjs"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const assetModule = join(frontendRoot, "src", "app", "viewer", "boardsAsset.ts")

type Kind = keyof BoardFiles

const EXTENSIONS: Record<Kind, string> = {runs: "bin", borders: "bin", capitals: "json", bridges: "json", terrain: "bin", provinces: "bin"}

// In the order a manifest names them.
const KINDS: readonly Kind[] = ["runs", "borders", "capitals", "bridges", "terrain", "provinces"]

// Every board has these; a board has provinces only once `npm run provinces` cuts them.
const REQUIRED: readonly Kind[] = ["runs", "borders", "capitals", "bridges", "terrain"]

const MANIFEST = /^board-[0-9a-f]{8}\.json$/

// A file of /map's board, as `npm run map` copies it: a stale one goes with the next copy.
const MAP_FILE = /^(board|runs|borders|capitals|bridges|terrain|provinces)-[0-9a-f]{8}\.(json|bin)$/

const PAGE_EXTENSIONS: Record<keyof PageFiles, string> = {borderLines: "bin", landmassNames: "json", provinceNames: "json"}

const PAGE_FILES: Record<keyof PageFiles, RegExp> = {
    borderLines: /^borderLines-[0-9a-f]{8}\.bin$/,
    landmassNames: /^landmassNames-[0-9a-f]{8}\.json$/,
    provinceNames: /^provinceNames-[0-9a-f]{8}\.json$/,
}

// One file of a board, named by its content; the one it replaces goes. The manifest names it once rewritten.
export function writeBoardFile(board: string, kind: Kind, bytes: Uint8Array): string {
    const fileName = `${kind}-${hashOf(bytes)}.${EXTENSIONS[kind]}`
    const directory = boardDir(board)
    sweep(directory, new RegExp(`^${kind}-[0-9a-f]{8}\\.${EXTENSIONS[kind]}$`), fileName)
    writeFileSync(join(directory, fileName), bytes)
    return fileName
}

export function writeBoardJSON(board: string, kind: "capitals" | "bridges", value: unknown): {fileName: string, byteLength: number} {
    const bytes = Buffer.from(`${JSON.stringify(value)}\n`)
    return {fileName: writeBoardFile(board, kind, bytes), byteLength: bytes.byteLength}
}

// The manifest over the files in the board's directory: what `fields` gives, and what the manifest held for the rest.
export function writeManifest(board: string, fields: Partial<Omit<Board, "files">> = {}): string {
    const directory = boardDir(board)
    const held = existsSync(directory) && readdirSync(directory).some((entry) => MANIFEST.test(entry))
        ? readManifest(directory).manifest
        : undefined
    const value = <K extends keyof Board>(key: K): Board[K] => {
        const given = (fields as Partial<Board>)[key] ?? held?.[key]
        if (given === undefined) throw new Error(`${board}'s manifest has no ${key}`)
        return given
    }
    const lattice = value("lattice")

    // A board written the first time has no bridges nor terrain until `npm run bridges` and `npm run terrain` write
    // them: its manifest leaves them out, and `syncBoards` copies it nowhere until it names every file.
    const files: Partial<BoardFiles> = {}
    for (const kind of KINDS) {
        const pattern = new RegExp(`^${kind}-[0-9a-f]{8}\\.${EXTENSIONS[kind]}$`)
        if (readdirSync(directory).some((entry) => pattern.test(entry))) files[kind] = blobNamed(pattern, directory)
    }
    const manifest = {
        name: board,
        body: value("body"),
        lattice: {cols: lattice.cols, warp: lattice.warp, corners: lattice.corners, order: lattice.order},
        tiles: value("tiles"),
        spacing: value("spacing"),
        positionsHash: value("positionsHash"),
        files,
    }
    const bytes = Buffer.from(`${JSON.stringify(manifest, null, 2)}\n`)
    const fileName = `board-${hashOf(bytes)}.json`
    sweep(directory, MANIFEST, fileName)
    writeFileSync(join(directory, fileName), bytes)
    return fileName
}

// Copies every board in /map to static/boards, the manifest and the files it names, and names them for the page. The
// page's own files of each board stay: /map does not carry them.
export function syncBoards(): {name: string, manifest: string}[] {
    const boards = directoriesIn(boardsDir)
    if (boards.length === 0) throw new Error(`no board in ${boardsDir}`)
    const copiesDir = join(staticDir, "boards")
    for (const gone of directoriesIn(copiesDir).filter((name) => !boards.includes(name))) {
        rmSync(join(copiesDir, gone), {recursive: true, force: true})
    }

    const whole = boards.filter((name) => {
        const missing = REQUIRED.filter((kind) => readManifest(boardDir(name)).manifest.files[kind] === undefined)
        if (missing.length > 0) console.warn(`map/boards/${name} has no ${missing.join(" nor ")} yet: copied nowhere until it has`)
        return missing.length === 0
    })
    const synced = whole.map((name) => {
        const from = boardDir(name)
        const {file, manifest} = readManifest(from)
        if (manifest.name !== name) throw new Error(`${from}/${file} names board ${manifest.name}`)

        const to = join(copiesDir, name)
        const copied = [file, ...Object.values(manifest.files)]
        mkdirSync(to, {recursive: true})
        for (const entry of readdirSync(to)) {
            if (MAP_FILE.test(entry) && !copied.includes(entry)) unlinkSync(join(to, entry))
        }
        for (const fileName of copied) copyFileSync(join(from, fileName), join(to, fileName))
        return {name, manifest: file}
    })
    writeBoardsTable()
    return synced
}

// One of the page's own files of a board, named by its content, in its copy; the one it replaces goes. The table names
// it once rewritten.
export function writePageFile(board: string, kind: keyof PageFiles, bytes: Uint8Array): string {
    const fileName = `${kind}-${hashOf(bytes)}.${PAGE_EXTENSIONS[kind]}`
    const directory = join(staticDir, "boards", board)
    sweep(directory, PAGE_FILES[kind], fileName)
    writeFileSync(join(directory, fileName), bytes)
    return fileName
}

// The page's table of every board it carries: a board's manifest, its own files and where each is served. A board
// missing one of its own files is left out, and says so: the page cannot play it until it has them.
export function writeBoardsTable(): string[] {
    const copiesDir = join(staticDir, "boards")
    const table: Record<string, BoardAsset> = {}
    for (const name of directoriesIn(copiesDir)) {
        const directory = join(copiesDir, name)
        const {manifest} = readManifest(directory)
        const needed: (keyof PageFiles)[] = ["borderLines", "landmassNames", ...(manifest.files.provinces ? ["provinceNames" as const] : [])]
        const missing = needed.filter((kind) => readdirSync(directory).filter((entry) => PAGE_FILES[kind].test(entry)).length !== 1)
        if (missing.length > 0) {
            const commands = missing.map((kind) => kind === "provinceNames" ? "provinces" : kind)
            console.warn(`static/boards/${name} has no ${missing.join(" nor ")}: run \`npm run ${commands.join("` and `npm run ")} -- ${name}\`; `
                + `the page cannot play it until then`)
            continue
        }
        const page: PageFiles = {
            borderLines: blobNamed(PAGE_FILES.borderLines, directory),
            landmassNames: blobNamed(PAGE_FILES.landmassNames, directory),
            ...(manifest.files.provinces ? {provinceNames: blobNamed(PAGE_FILES.provinceNames, directory)} : {}),
        }
        const urls = Object.fromEntries(Object.entries({...manifest.files, ...page})
            .map(([kind, fileName]) => [kind, `/static/boards/${name}/${fileName}`])) as BoardFiles & PageFiles
        table[name] = {...manifest, page, urls}
    }
    writeFileSync(assetModule, `// Generated by \`npm run map\`, \`npm run borderLines\`, \`npm run landmassNames\` and \`npm run provinces\` from /map/boards and
// static/boards: do not edit.
import type {BoardAsset} from "../../domain/board.ts"

export const BOARDS: Readonly<Record<string, BoardAsset>> = ${JSON.stringify(table, null, 4)}
`)
    return Object.keys(table)
}

// The boards in /map, by name: `npm run borderLines -- earth-1` asks for one, and none for every board.
export function boardsAsked(args: readonly string[]): string[] {
    const boards = directoriesIn(boardsDir)
    const asked = args.filter((arg) => !arg.startsWith("-"))
    for (const name of asked) if (!boards.includes(name)) throw new Error(`no board ${name} in ${boardsDir}: it has ${boards.join(", ")}`)
    return asked.length > 0 ? asked : boards
}

function directoriesIn(directory: string): string[] {
    if (!existsSync(directory)) return []
    return readdirSync(directory, {withFileTypes: true}).filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort()
}

function hashOf(bytes: Uint8Array): string {
    return createHash("sha256").update(bytes).digest("hex").slice(0, 8)
}

function sweep(directory: string, pattern: RegExp, keep: string): void {
    mkdirSync(directory, {recursive: true})
    for (const entry of readdirSync(directory)) {
        if (pattern.test(entry) && entry !== keep) unlinkSync(join(directory, entry))
    }
}
