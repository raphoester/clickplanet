import countriesData from "../../static/countries/countries.json";

export const Countries: Map<string, Country> = convertToMap(countriesData);

export type Country = {
    name: string,
    code: string
}

function convertToMap(data: Record<string, string>): Map<string, { name: string; code: string }> {
    const resultMap = new Map<string, Country>();
    for (const [code, name] of Object.entries(data)) {
        resultMap.set(code, {name, code});
    }
    return resultMap;
}

// countries.json carries each flag as an emoji in front of the name. Windows has
// no glyph for those, and draws a box with the country code in it, so anything
// that shows the flag itself has to take the emoji back off the name.
const LEADING_FLAG = /^(?:\p{Regional_Indicator}{2}|\p{Extended_Pictographic}[\u{E0020}-\u{E007F}]*\uFE0F?)\s*/u

export function nameWithoutFlag(country: Country): string {
    return country.name.replace(LEADING_FLAG, "")
}
