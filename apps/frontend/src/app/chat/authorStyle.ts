import {CSSProperties} from "react";
import {authorHue} from "../../domain/authorColor.ts";

export function authorStyle(authorName: string): CSSProperties {
    return {"--author-hue": authorHue(authorName)} as CSSProperties
}
