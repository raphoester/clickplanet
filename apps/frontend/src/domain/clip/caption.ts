// credit: what the licence of the clip's music asks to be said, where the clip is posted.
export function credited(caption: string, credit: string | undefined): string {
    return credit === undefined ? caption : `${caption}\n🎵 ${credit}`
}
