/**
 * What `assets.binding` in wrangler.jsonc hands the Worker. `wrangler types`
 * would generate this along with 15 000 lines of runtime types; the runtime
 * types come from `@cloudflare/workers-types` instead, so the only part worth
 * keeping is the binding itself.
 */
interface Env {
    ASSETS: Fetcher
}
