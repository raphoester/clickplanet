import {Language} from "./languages.ts"

// The display face of a language Luckiest Guy cannot draw (tokens.css names it), in its one heavy weight: a page asks
// for any weight and gets that one. Only the player's language is downloaded.
const DISPLAY_FACES: Readonly<Record<Language, string | null>> = {
    "en": null,
    "fr": null,
    "de": null,
    "tr": null,
    "pt-BR": null,
    "pl": null,
    "id": null,
    "es": null,
    "cs": null,
    "sv": null,
    "hu": null,
    "ms": null,
    "ru": "Nunito:wght@900",
    "bg": "Nunito:wght@900",
    "vi": "Baloo+2:wght@800",
    "hi": "Baloo+2:wght@800",
    "ar": "Baloo+Bhaijaan+2:wght@800",
    "ja": "M+PLUS+Rounded+1c:wght@800",
    "zh-Hans": "Noto+Sans+SC:wght@900",
}

export function loadDisplayFace(language: Language) {
    const face = DISPLAY_FACES[language]
    if (face === null) return
    const link = document.createElement("link")
    link.rel = "stylesheet"
    link.href = `https://fonts.googleapis.com/css2?family=${face}&display=swap`
    document.head.append(link)
}
