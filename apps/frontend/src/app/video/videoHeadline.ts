import {percentOf, VideoStory} from "../../domain/video/story.ts"
import {t, translator} from "../../i18n/page.ts"

// What the moment and the share say of the flag's day, in capitals as the video says it.
export function videoHeadline(story: VideoStory): string {
    const flag = translator().country(story.flag)
    const country = translator().country(story.country)
    const share = (part: number) => translator().number(percentOf(part, story.tiles) / 100, {style: "percent"})
    return lineOf(story, flag, country, share).toLocaleUpperCase(translator().locale)
}

function lineOf(story: VideoStory, flag: string, country: string, share: (part: number) => string): string {
    switch (story.kind) {
        case "gain":
            return t("video.headline.gain", {flag, country, share: share(story.end - story.start)})
        case "loss":
            return t("video.headline.loss", {flag, country})
        case "quiet":
            return t("video.headline.quiet", {flag, country, share: share(story.end)})
    }
}
