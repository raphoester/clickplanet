// Every experiment the generator knows, and the arms of each. Which experiments run, and which arm a clip is in, are
// the job's (apps/backend/docs/clips-job.md): it flips the coins and hands the arms over with the pick. An
// arm this app acts on is applied when the clip is rendered; an experiment that acts only when the clip is posted is
// listed here too, with nothing reading it, so the generator still refuses an arm nobody knows and carries the rest to
// the job in `ClipRecord.variants`.
export const EXPERIMENTS = {
    // as-chosen: as long as its camera has somewhere to go (`pace.ts`). short: the shortest a story of its kind is.
    length: ["as-chosen", "short"],
    // Which stories a run tells and their headlines, decided by the job before it chooses, once per run: rules, as
    // before, or a model choosing among what the rules let through and writing each headline, which reaches this app
    // as --headline. fell-back is a run the model was drawn for that the rules made, its answer missing or refused.
    // Nothing to apply here: the headline is a flag of its own.
    editor: ["rules", "model", "fell-back"],
    // TikTok's sound, applied by the job when it posts: anthem, as before, or a track trending on TikTok attached over
    // the muted video. Nothing to apply here: the video is the same in both, anthem included. The job's own fell-back
    // (a trending clip posted with the anthem, the chart unread) is decided after the render, so it never reaches this
    // app and is refused here like any arm the job does not draw.
    sound: ["anthem", "trending"],
} as const satisfies Record<string, readonly string[]>

type Experiments = typeof EXPERIMENTS

export type Variants = {readonly [Name in keyof Experiments]?: Experiments[Name][number]}

const known = (name: string): name is keyof Experiments => Object.hasOwn(EXPERIMENTS, name)

// written: the arms as the job wrote them, a JSON object of words, `{"length":"short"}`; none is no experiment. An
// experiment or an arm this app does not know fails the clip: a clip made in the wrong arm would be counted in it.
export function variantsOf(written: string | null): Variants {
    if (written === null || written.trim() === "") return {}

    const parsed: unknown = JSON.parse(written)
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
        throw new Error(`the variants ${written} are not an object of an arm by experiment`)
    }
    for (const [name, arm] of Object.entries(parsed)) {
        if (!known(name)) throw new Error(`no experiment "${name}"`)
        const arms: readonly string[] = EXPERIMENTS[name]
        if (typeof arm !== "string" || !arms.includes(arm)) {
            throw new Error(`no arm ${JSON.stringify(arm)} in the experiment "${name}": ${arms.join(", ")}`)
        }
    }
    return parsed as Variants
}
