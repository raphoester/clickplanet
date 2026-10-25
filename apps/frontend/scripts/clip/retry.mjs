// What a launch that fails now and then is given before the run fails: Chrome once never exposed its debugging target
// in 35 runs of the clips workflow (2026-10-24), and the next run passed. A lost run is a lost chance to post on time.
export const LAUNCH_TRIES = 3

// start(attempt) until it answers, at most tries times; each failure but the last is told to log, then given back.
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
