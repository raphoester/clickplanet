import * as fs from "node:fs";
import * as path from "node:path";
import * as Smith from "spritesmith";
import {writeAtlas} from "./writeAtlas.ts";

const args = process.argv.slice(2);
const spritesDir = args[0] || './static/countries/png100px';

const pngFiles = fs.readdirSync(spritesDir)
    .filter(file => file.endsWith('.png'))
    .map(file => path.join(spritesDir, file));

Smith.default.run({src: pngFiles}, (err: Error | null, result: Smith.SpritesmithResult) => {
    if (err) throw err;

    const {fileName, width, height} = writeAtlas(result.image, result.coordinates);
    console.log(`Atlas generated: ${fileName} (${width}x${height}, ${pngFiles.length} flags)`);
});
