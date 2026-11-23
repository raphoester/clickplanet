export type Env = {
    MAILROOM_URL: string
    MAILROOM_SECRET: string
}

export type Mail = Pick<ForwardableEmailMessage, "to" | "raw" | "setReject">

export async function deliver(mail: Mail, env: Env): Promise<void> {
    const res = await fetch(env.MAILROOM_URL, {
        method: "POST",
        headers: {
            "Authorization": `Bearer ${env.MAILROOM_SECRET}`,
            "Content-Type": "message/rfc822",
            "X-Envelope-To": mail.to,
        },
        body: await new Response(mail.raw).arrayBuffer(),
    })
    if (res.status === 413) {
        mail.setReject("This mail is too large.")
        return
    }
    // A throw answers the sending server a temporary failure: it tries again later.
    if (!res.ok) throw new Error(`the backend answered ${res.status}`)
}

export default {
    async email(message: ForwardableEmailMessage, env: Env): Promise<void> {
        await deliver(message, env)
    },
} satisfies ExportedHandler<Env>
