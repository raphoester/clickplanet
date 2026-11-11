import {CapturedFrame} from "../viewer/capture.ts";
import {ShareAddress, shareFileName, ShareStats} from "../../domain/shareCard.ts";
import {drawShareCard} from "./drawShareCard.ts";

export async function takePicture(
    capture: () => Promise<CapturedFrame>,
    stats: ShareStats,
    address: ShareAddress,
): Promise<File> {
    const frame = await capture()
    const image = await drawShareCard(frame, stats, address)

    return new File([image], shareFileName(stats.country.code), {type: "image/png"})
}
