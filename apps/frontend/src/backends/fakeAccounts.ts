// One account per name, so the fakes agree on who a chat message, a roster line and a standing are.
export function fakeAccountId(name: string): string {
    return `account-${name}`
}
