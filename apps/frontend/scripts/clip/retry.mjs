export const LAUNCH_TRIES = 3

export async function withRetries(start, {tries, log}) {
    for (let attempt = 1; ; attempt++) {
        try {
            return await start(attempt)
        } catch (error) {
            if (attempt >= tries) throw error
            log(`${error.message}: try ${attempt + 1} of ${tries}`)
        }
    }
}
