export function newMessages(unread: number): string {
    return unread === 1 ? "1 new message" : `${unread} new messages`
}
