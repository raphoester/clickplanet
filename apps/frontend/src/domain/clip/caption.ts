export function credited(caption: string, credit: string | undefined): string {
    return credit === undefined ? caption : `${caption}\n🎵 ${credit}`
}
