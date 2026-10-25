// One account per name, so the fakes agree on who a chat message, a roster line and a standing are.
export function fakeAccountId(name: string): string {
    return `account-${name}`
}

export const FAKE_TAKERS = ["Ana", "kiran_07", "Mateus", "zoe_nz", "guest_91aa3d", "guest_3b7f02"].map(fakeAccountId)

export const FAKE_YOU = fakeAccountId("you")
