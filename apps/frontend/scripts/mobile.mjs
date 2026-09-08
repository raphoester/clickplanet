/*
 * Screenshots and inspects the app as a phone, using Chrome's device emulation.
 *
 * The point is that resizing a window does not work: macOS enforces a minimum
 * window width well above the 768px breakpoint, so the mobile branch of the CSS
 * never engages and you end up checking a narrow desktop instead. This drives
 * the same DevTools Protocol calls the inspector's device toolbar makes —
 * Emulation.setDeviceMetricsOverride, setTouchEmulationEnabled,
 * setUserAgentOverride — so the viewport, DPR, `pointer: coarse` and the iOS
 * user agent all resolve the way they do on the device.
 *
 * Headless Chrome with a throwaway profile, so it never touches the browser you
 * are using. No dependencies: Node's global WebSocket does the CDP talking.
 *
 *   node scripts/mobile.mjs <url> [options]
 *
 *   --out <path>    write a PNG there
 *   --eval <expr>   evaluate in the page and print the result (await supported)
 *   --open-menu     dismiss the donation modal and unfold the menu first
 *   --w --h --dpr   viewport, default iPhone 14 (390 × 844 @3)
 *   --full          capture the whole page rather than the viewport
 *   --settle <ms>   wait after load before acting, default 1500
 *   --headed        show the window (for watching an interaction go wrong)
 *
 * CHROME_PATH overrides the browser binary.
 */
import {spawn} from "node:child_process";
import {mkdirSync, readFileSync, rmSync, writeFileSync} from "node:fs";
import {dirname, join} from "node:path";
import {tmpdir} from "node:os";

const args = process.argv.slice(2);
const url = args[0];
const flag = (name, fallback) => {
    const i = args.indexOf(`--${name}`);
    return i === -1 ? fallback : args[i + 1];
};
const has = (name) => args.includes(`--${name}`);

if (!url || url.startsWith("--")) {
    console.error("usage: node scripts/mobile.mjs <url> [--out shot.png] [--eval expr] [--open-menu]");
    process.exit(1);
}

const CHROME = process.env.CHROME_PATH ?? "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const PROFILE = join(tmpdir(), `clickplanet-mobile-${process.pid}`);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const width = Number(flag("w", 390));
const height = Number(flag("h", 844));

const chrome = spawn(CHROME, [
    ...(has("headed") ? [] : ["--headless=new"]),
    // Port 0 asks Chrome to pick a free one and write it into the profile, so
    // two of these can run at once and neither collides with a real browser.
    "--remote-debugging-port=0",
    `--user-data-dir=${PROFILE}`,
    "--no-first-run",
    "--no-default-browser-check",
    // The globe needs WebGL, which headless has to software-render.
    "--enable-unsafe-swiftshader",
    "--hide-scrollbars",
    "about:blank",
], {stdio: "ignore"});

const done = (code) => {
    chrome.kill();
    // Chrome keeps writing to the profile as it shuts down, so the first rm
    // races it. A leftover temp directory is not worth failing the run over.
    try {
        rmSync(PROFILE, {recursive: true, force: true, maxRetries: 20, retryDelay: 100});
    } catch { /* it is under tmpdir; the OS will get it */ }
    process.exit(code);
};
process.on("SIGINT", () => done(130));

async function debuggerUrl() {
    for (let i = 0; i < 80; i++) {
        try {
            const port = readFileSync(join(PROFILE, "DevToolsActivePort"), "utf8").split("\n")[0];
            const targets = await fetch(`http://127.0.0.1:${port}/json/list`).then((r) => r.json());
            const page = targets.find((t) => t.type === "page");
            if (page) return page.webSocketDebuggerUrl;
        } catch { /* Chrome is still starting */ }
        await sleep(250);
    }
    throw new Error("Chrome never exposed a debugging target");
}

const ws = new WebSocket(await debuggerUrl());
await new Promise((r) => ws.addEventListener("open", r, {once: true}));

let lastId = 0;
const pending = new Map();
const seen = new Set();
ws.addEventListener("message", (m) => {
    const msg = JSON.parse(m.data);
    if (msg.id && pending.has(msg.id)) {
        const {resolve, reject} = pending.get(msg.id);
        pending.delete(msg.id);
        msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
    } else if (msg.method) {
        seen.add(msg.method);
    }
});
const send = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++lastId;
    pending.set(id, {resolve, reject});
    ws.send(JSON.stringify({id, method, params}));
});

const evaluate = async (expression) => {
    const {result, exceptionDetails} = await send("Runtime.evaluate", {
        expression, returnByValue: true, awaitPromise: true,
    });
    if (exceptionDetails) throw new Error(exceptionDetails.exception?.description ?? exceptionDetails.text);
    return result.value;
};

try {
    await send("Page.enable");
    await send("Runtime.enable");
    await send("Emulation.setDeviceMetricsOverride", {
        width, height,
        deviceScaleFactor: Number(flag("dpr", 3)),
        mobile: true,
        screenWidth: width,
        screenHeight: height,
    });
    await send("Emulation.setTouchEmulationEnabled", {enabled: true, maxTouchPoints: 5});
    await send("Emulation.setEmitTouchEventsForMouse", {enabled: true, configuration: "mobile"});
    await send("Emulation.setUserAgentOverride", {
        userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 " +
            "(KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
        platform: "iPhone",
    });

    await send("Page.navigate", {url});
    for (let i = 0; i < 60 && !seen.has("Page.loadEventFired"); i++) await sleep(150);
    await sleep(Number(flag("settle", 1500)));

    if (has("open-menu")) {
        // Two things stand between a fresh load and the menu: the donation
        // modal, which rolls a coin on every load, and the fold, which starts
        // closed on mobile.
        await evaluate(`(async () => {
            document.querySelector("[role=dialog] button")?.click()
            const toggle = document.querySelector(".menu-collapse")
            if (toggle?.getAttribute("aria-expanded") !== "true") toggle?.click()
            await new Promise(r => setTimeout(r, 600))
        })()`);
    }

    const expression = flag("eval", null);
    if (expression) {
        const value = await evaluate(expression);
        console.log(typeof value === "string" ? value : JSON.stringify(value));
    }

    const out = flag("out", null);
    if (out) {
        const {data} = await send("Page.captureScreenshot", {
            format: "png",
            captureBeyondViewport: has("full"),
        });
        mkdirSync(dirname(out), {recursive: true});
        writeFileSync(out, Buffer.from(data, "base64"));
        console.log(`saved ${out}`);
    }
} catch (error) {
    console.error(error.message);
    done(1);
}

done(0);
