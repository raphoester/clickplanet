import {flagCatalog} from "./flagsStore.ts"

// A flag as one image file: a sold flag's own picture, a built one's SVG.
export function flagPictureUrl(code: string): string {
    return flagCatalog().pictureOf(code)?.url ?? `/static/countries/svg/${code}.svg`
}
