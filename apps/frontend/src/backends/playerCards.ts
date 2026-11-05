import {PlayerInfo, PlayerInfoBackend, PlayerReader} from "./player.ts"
import {NO_PROGRESS, ProgressReader} from "./progression.ts"

export class PlayerCards implements PlayerInfoBackend {
    constructor(
        private readonly players: PlayerReader,
        private readonly progression: ProgressReader,
    ) {
    }

    public async playerInfo(accountId: string): Promise<PlayerInfo | undefined> {
        const [player, progress] = await Promise.all([
            this.players.player(accountId),
            this.progression.progress(accountId),
        ])
        if (!player) return undefined
        if (player.guest) return {...player, ...NO_PROGRESS}
        return {...player, ...(progress ?? NO_PROGRESS)}
    }
}
