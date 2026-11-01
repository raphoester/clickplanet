export const ARCHIVE_PARAM = "archive"

export function archiveUrlOf(season: number): string {
    return `/play?${ARCHIVE_PARAM}=${season}`
}
