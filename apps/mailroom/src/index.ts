import {deliver, Env} from "./deliver.ts"

export default {
    async email(message: ForwardableEmailMessage, env: Env): Promise<void> {
        await deliver(message, env)
    },
} satisfies ExportedHandler<Env>
