import {CapturedFrame} from "../viewer/capture.ts";
import {shareFileName, ShareStats} from "../../domain/shareCard.ts";
import {drawShareCard} from "./drawShareCard.ts";

export async function takePicture(
    capture: () => Promise<CapturedFrame>,
    stats: ShareStats,
): Promise<File> {
    const frame = await capture()
    const image = await drawShareCard(frame, stats)

    return new File([image], shareFileName(stats.country.code), {type: "image/png"})
}
