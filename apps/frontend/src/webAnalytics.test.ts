import {describe, expect, it} from "vitest"
import home from "../index.html?raw"
import play from "../play.html?raw"
import privacy from "../privacy.html?raw"
import terms from "../terms.html?raw"
import {WEB_ANALYTICS_TOKEN, withoutAnalytics} from "./webAnalytics.ts"

const BEACON = `<script type="module" src="https://static.cloudflareinsights.com/beacon.min.js" data-cf-beacon='{"token": "%WEB_ANALYTICS_TOKEN%", "spa": false}'></script>`

describe("Cloudflare Web Analytics", () => {
    for (const [page, html] of Object.entries({home, play, privacy, terms})) {
        it(`loads on the ${page} page, its token from webAnalytics.ts`, () => {
            expect(html).toContain(BEACON)
            expect(html).not.toContain(WEB_ANALYTICS_TOKEN)
        })
    }

    it("stays off the sign-in callback, which still runs the game", () => {
        const callback = withoutAnalytics(play.replace("%WEB_ANALYTICS_TOKEN%", WEB_ANALYTICS_TOKEN))

        expect(callback).not.toContain("cloudflareinsights")
        expect(callback).toContain(`<script type="module" src="/src/main.tsx"></script>`)
    })
})
