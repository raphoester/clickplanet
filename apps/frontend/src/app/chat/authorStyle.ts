import {CSSProperties} from "react";
import {authorHue} from "../../domain/authorColor.ts";

/**
 * The author's colour, handed to the CSS as a hue only: `ChatPanel.css` fixes
 * the saturation and the lightness so every author reads the same against the
 * dark panel.
 */
export function authorStyle(authorName: string): CSSProperties {
    return {"--author-hue": authorHue(authorName)} as CSSProperties
}
