import {cardForQuery, metaOverrides} from "../src/domain/shareCard.ts"

/**
 * The production entry point. Everything it does is for the crawlers: a scraper
 * building a link preview does not run our JavaScript, so a per-country card
 * has to be in the HTML bytes we hand back, not in anything React sets later.
 *
 * It sits in front of the static assets rather than replacing them —
 * `env.ASSETS.fetch` does exactly what the asset handler did before this Worker
 * existed, including the single-page-application fallback — and it only ever
 * rewrites the `content` of meta tags that are already in `index.html`. That is
 * what keeps the nginx image in `deploy/` honest: it serves the same
 * `index.html` with no Worker in front of it, and gets the generic card.
 *
 * `run_worker_first` in wrangler.jsonc keeps `/assets/*` and `/static/*` out of
 * here entirely, so the bundles, the textures and the coordinates blob are
 * served by the asset store without ever waking this up.
 */
export default {
    async fetch(request: Request, env: Env): Promise<Response> {
        // Only something you can put in an address bar can be shared, and only
        // a document carries the tags. Everything else goes back exactly as the
        // asset handler produced it.
        const wanted = request.method === "GET" || request.method === "HEAD"
            ? cardForQuery(new URL(request.url).search)
            : undefined

        // No `?c=`, a code we have no card for, or a malformed one: hand back
        // the asset untouched, so a plain https://clickplanet.lol/ is byte for
        // byte what it was before this Worker existed.
        if (!wanted) return env.ASSETS.fetch(request)

        // Every copy of index.html carries the same ETag, and that stops being
        // true the moment the tags in it differ per country. Asking
        // unconditionally is what stops the asset handler answering 304 to a
        // scraper — or to a browser holding the generic document from before
        // this Worker shipped — and leaving it on a card for the wrong country,
        // or no country at all.
        const unconditional = new Request(request)
        unconditional.headers.delete("If-None-Match")
        unconditional.headers.delete("If-Modified-Since")

        const response = await env.ASSETS.fetch(unconditional)
        if (!isHtmlDocument(response)) return response

        // HTMLRewriter streams, so the document is never held in memory, and
        // `setAttribute` escapes what it writes. The card is built from the
        // country list either way — the query string itself never reaches it.
        const rewritten = metaOverrides(wanted)
            .reduce(
                (rewriter, {selector, content}) => rewriter.on(selector, new SetContent(content)),
                new HTMLRewriter(),
            )
            .transform(response)

        // And drop the ETag on the way out for the same reason: it belongs to
        // the document the asset handler produced, not to the one being sent.
        // The page is `max-age=0, must-revalidate` anyway, so nothing that was
        // worth having is lost with it.
        const headers = new Headers(rewritten.headers)
        headers.delete("ETag")

        return new Response(rewritten.body, {
            status: rewritten.status,
            statusText: rewritten.statusText,
            headers,
        })
    },
} satisfies ExportedHandler<Env>

class SetContent {
    constructor(private readonly content: string) {
    }

    element(element: Element) {
        element.setAttribute("content", this.content)
    }
}

function isHtmlDocument(response: Response): boolean {
    // `text/html; charset=utf-8`, and whatever casing the asset handler picks.
    return response.headers.get("content-type")?.toLowerCase().startsWith("text/html") ?? false
}
