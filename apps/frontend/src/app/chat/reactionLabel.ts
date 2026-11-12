import {Reaction} from "../../gen/grpc/chat/v1/chat_pb.ts"
import {Key} from "../../i18n/translator.ts"

// A reaction's line is named after its proto value: THUMBS_UP is `reaction.thumbsUp`.
export function reactionLabelOf(reaction: Reaction): Key {
    const words = Reaction[reaction].toLowerCase().split("_")
    return `reaction.${words[0]}${words.slice(1).map((word) => word[0].toUpperCase() + word.slice(1)).join("")}` as Key
}
