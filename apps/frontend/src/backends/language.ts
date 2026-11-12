import {Interceptor} from "@connectrpc/connect"

// Every call says the page's language; a safelisted header, so it costs no CORS preflight.
export function languageInterceptor(language: Promise<string>): Interceptor {
    return (next) => async (req) => {
        req.header.set("Accept-Language", await language)
        return next(req)
    }
}
