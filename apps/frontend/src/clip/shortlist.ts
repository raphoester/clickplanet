import {Shortlist} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Story, storyCountriesOf, storyKeyOf} from "../domain/clip/story.ts"
import {regionOf} from "../domain/regions.ts"
import {wordsOf} from "./overlay.ts"

export function shortlistOf(stories: readonly Story[]): Shortlist {
    return new Shortlist({
        stories: stories.map((story) => ({
            title: wordsOf(story).headline, key: storyKeyOf(story, regionOf), countries: storyCountriesOf(story),
        })),
    })
}
