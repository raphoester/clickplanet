export const WEB_ANALYTICS_TOKEN = "db7aa79ae8e44b89b9b44e73ea07808a"

const BEACON = /\s*<script type="module" src="https:\/\/static\.cloudflareinsights\.com\/beacon\.min\.js"[^>]*><\/script>/g

export function withoutAnalytics(page: string): string {
    return page.replace(BEACON, "")
}
