import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {FlagService} from "../gen/grpc/flags/v1/flags_connect.ts"
import type {PartialMessage} from "@bufbuild/protobuf"
import {Anthem, Flag, FlagEvent, Picture as PicturePb, Pin as PinPb} from "../gen/grpc/flags/v1/flags_pb.ts"
import type {FlagPicture, ListedFlag, Pin} from "../domain/flags.ts"
import {FlagHandlers, FlagLister, FlagStream} from "./flags.ts"
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts"

// The fresh client asks past the browser's copy, which may predate the change it reads for.
export function newFlagServiceClient(config: Config, fresh = false): PromiseClient<typeof FlagService> {
    return createPromiseClient(FlagService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
        fetch: fresh ? (input, init) => globalThis.fetch(input, {...init, cache: "no-cache"}) : undefined,
    }))
}

export function flagChangedOf(event: FlagEvent): ListedFlag | undefined {
    return event.event.case === "flagChanged" ? listedOf(event.event.value) : undefined
}

// A sold flag with no picture, which no server sends, is listed as an opened flag with none: the blank.
export function listedOf({code, name, closed, sold}: Flag): ListedFlag {
    const picture = sold?.picture
    if (!sold || !picture || picture.url === "") return {code, name, closed}
    return {
        code, name, closed,
        sold: {
            picture: pictureOf(picture),
            endsAt: sold.endsAt ? sold.endsAt.toDate().getTime() : 0,
            closedAt: sold.closedAt ? sold.closedAt.toDate().getTime() : undefined,
            ...sold.pin ? {pin: pinOf(sold.pin)} : {},
            ...sold.removed ? {removed: true} : {},
            ...anthemPart(sold.anthem),
        },
    }
}

export function pinOf({latitude, longitude}: PinPb): Pin {
    return {latitude, longitude}
}

export function pictureOf(picture: PicturePb): FlagPicture {
    return {
        url: picture.url, width: picture.width, height: picture.height,
        stretch: picture.stretch, focusX: picture.focusX, focusY: picture.focusY,
    }
}

// A server from before the flags module has neither the list nor the stream.
const unimplemented = (e: unknown) => e instanceof ConnectError && e.code === Code.Unimplemented

export class ConnectFlagsBackend implements FlagLister, FlagStream {
    constructor(
        private readonly client: PromiseClient<typeof FlagService>,
        private readonly fresh: PromiseClient<typeof FlagService>,
    ) {
    }

    public async listFlags(fresh: boolean): Promise<ListedFlag[]> {
        const client = fresh ? this.fresh : this.client
        try {
            const res = await retrying(() => client.listFlags({}), "ListFlags")
            return res.flags.map(listedOf)
        } catch (e) {
            if (unimplemented(e)) return []
            throw e
        }
    }

    public listenForFlags({onOpened, onChanged}: FlagHandlers): () => void {
        let opening = false
        return openStream(
            (signal) => {
                opening = true
                return this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
                if (opening) {
                    opening = false
                    onOpened()
                }
                const flag = flagChangedOf(event)
                if (flag) onChanged(flag)
            },
            "flag events",
            {stopsOn: unimplemented},
        )
    }
}

// The anthem's field of a page's flag or submission: none for none.
export function anthemPart(anthem: Anthem | undefined): {anthem?: string} {
    const code = anthemOf(anthem)
    return code ? {anthem: code} : {}
}

// A recording of the game's library, by its code; none for a source this build does not know.
function anthemOf(anthem: Anthem | undefined): string | undefined {
    return anthem?.source.case === "libraryCode" && anthem.source.value !== "" ? anthem.source.value : undefined
}

// The wire's anthem: unset for none.
export function anthemMessageOf(code: string | undefined): PartialMessage<Anthem> | undefined {
    return code ? {source: {case: "libraryCode", value: code}} : undefined
}
