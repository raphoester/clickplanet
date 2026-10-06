import {spawn} from "node:child_process"
import {createWriteStream, readFileSync} from "node:fs"

const args = process.argv.slice(2)
const flag = (name, fallback) => {
    const i = args.indexOf(`--${name}`)
    return i === -1 ? fallback : args[i + 1]
}

const host = flag("ssh", process.env.CLICKPLANET_SSH)
const out = flag("out")
if (!host || !out || args.includes("--help")) {
    console.error(`usage: npm run clip:fetch -- --ssh <user@host> --out <replay.json> [--hours 72] [--until <ISO time>]

Asks the production backend's operator tool for a replay (planet.v1.AdminService/GetReplay), over SSH.
CLICKPLANET_SSH stands in for --ssh. The ledger keeps 72 hours in memory: a replay cannot reach further back.`)
    process.exit(1)
}

const until = flag("until") ? new Date(flag("until")) : new Date()
const since = new Date(until.getTime() - Number(flag("hours", "72")) * 3_600_000)
const body = JSON.stringify({since: since.toISOString(), until: until.toISOString()})

const remote = [
    "cd /opt/clickplanet/deploy/vps &&",
    "docker compose exec -T backend wget -qO-",
    "--header 'Content-Type: application/json'",
    `--post-data '${body}'`,
    "http://127.0.0.1:8081/planet.v1.AdminService/GetReplay",
].join(" ")

const ssh = spawn("ssh", [host, remote], {stdio: ["ignore", "pipe", "inherit"]})
const file = createWriteStream(out)
ssh.stdout.pipe(file)

const [code] = await Promise.all([
    new Promise((resolve) => ssh.on("exit", resolve)),
    new Promise((resolve) => file.on("close", resolve)),
])
if (code !== 0) {
    console.error(`ssh exited with ${code}`)
    process.exit(1)
}

const replay = JSON.parse(readFileSync(out, "utf8"))
console.log(`saved ${replay.events?.length ?? 0} events from ${replay.since} to ${replay.until} in ${out}`)
