// A row a ground the flags are painted over: the landmasses', and the provinces'.
uniform sampler2D landmassData;
uniform float landmassCount;
uniform sampler2D provinceData;
uniform float provinceCount;
// How far the provinces divide the map: 0 the countries' landmasses, 1 the provinces, in between for the short fade
// when the zoom crosses the provinces' line (pointSize.ts).
uniform float provinceShare;
// Each province's landmass, PROVINCE_COLUMNS a row.
uniform highp usampler2D provinceLandmass;

uniform float flagPaint;
// 1: each tile paints its own holder's flag. 0: the landmass's leader paints all of it, at its share.
uniform float splitFlags;

// The provinces lit, first and last: a landmass's span, or one province.
uniform vec2 highlight;

uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform sampler2D slotTexture;
uniform sampler2D flagRectangles;
uniform sampler2D slotFrames;

// Must match latticeCells.ts, which finds a direction's cell the same way.
#define FACES 20
uniform vec4 faceNormals[FACES];
uniform sampler2D latticeFaces;
uniform int latticeCols;
// The board's warp of the barycentric weights (lattice.ts): a, b and c, and w 1 when the board has one, 0 for earth-1.
uniform vec4 latticeWarp;
uniform highp usampler2D tileCells;
uniform highp usampler2D tileData;
// Each tile's flag's footprint on its province, and a row a footprint: the provinces' first, then the landmasses'
// (footprintField.ts).
uniform highp usampler2D footprintData;
uniform sampler2D footprintFrames;
uniform highp usampler2D terrain;
// Must match KINDS in domain/terrain.ts.
uniform vec3 terrainColours[12];

// Must match CELL_COLUMNS and PROVINCE_COLUMNS in territoryLayer.ts, TILE_COLUMNS in tileField.ts and TERRAIN_COLUMNS
// in terrain.ts.
const int CELL_COLUMNS = 256;
const int PROVINCE_COLUMNS = 1024;
const int TILE_COLUMNS = 1024;
const int TERRAIN_COLUMNS = 2048;
// Must match FOOTPRINT_COLUMNS and MOST_FOOTPRINTS in footprintField.ts.
const int FOOTPRINT_COLUMNS = 1024;
const int FOOTPRINT_ROWS = 32;

// Must match domain/terrain.ts: the land's kinds, then the sea's, the last the open sea.
const uint LAND_KINDS = 8u;
const uint OPEN_SEA = 11u;

// Each cell of land is up to this much lighter or darker, so a patch of one kind is not one flat colour.
const float SHADE_JITTER = 0.04;
// A cell of land's edge is this much darker, drawn as a cell grows from EDGE_FROM to EDGE_FULL pixels wide.
const float CELL_EDGE = 0.14;
const float EDGE_FROM = 10.0;
const float EDGE_FULL = 18.0;

#include ../flagFrame.glsl;
#include ../tileEffects.glsl;
#include ../effectColours.glsl;

varying vec3 vWorld;

const float HIGHLIGHT_LIFT = 0.2;
const float HIGHLIGHT_FLOOR = 0.35;

// Must match MAX_FLAG_RADIUS in borderField.ts.
const float MAX_FLAG_RADIUS = 0.42;
// About a tile of the board the globe draws (globe.ts).
uniform float leastFlagRadius;

uint popcount(uint bits) {
    bits = bits - ((bits >> 1u) & 0x55555555u);
    bits = (bits & 0x33333333u) + ((bits >> 2u) & 0x33333333u);
    return (((bits + (bits >> 4u)) & 0x0F0F0F0Fu) * 0x01010101u) >> 24u;
}

ivec2 nearestOnGrid(vec2 at) {
    vec2 corner = floor(at);
    vec2 r = at - corner;
    if (r.x + r.y < 1.0) {
        float rest = 1.0 - r.x - r.y;
        if (rest >= r.x && rest >= r.y) return ivec2(corner);
        return ivec2(corner) + (r.x >= r.y ? ivec2(1, 0) : ivec2(0, 1));
    }
    float far = r.x + r.y - 1.0;
    float right = 1.0 - r.y;
    float up = 1.0 - r.x;
    if (far >= right && far >= up) return ivec2(corner) + 1;
    return ivec2(corner) + (right >= up ? ivec2(1, 0) : ivec2(0, 1));
}

// The warp of the weights (wa, wb, wc), as lattice.ts's warpedWeights.
vec3 warped(vec3 w) {
    float shared = latticeWarp.z * w.x * w.y * w.z;
    vec3 h = w * (1.0 + latticeWarp.x * w + latticeWarp.y * w * w) + shared;
    return h / (h.x + h.y + h.z);
}

// The weights (wb, wc) whose warp is `target`: two Newton steps from the target itself, as lattice.ts's unwarp.
vec2 unwarped(vec2 target) {
    float a = latticeWarp.x;
    float b = latticeWarp.y;
    float c = latticeWarp.z;
    vec2 uv = target;
    for (int step = 0; step < 2; step++) {
        float wa = 1.0 - uv.x - uv.y;
        vec3 w = vec3(wa, uv);
        vec3 h = w * (1.0 + a * w + b * w * w) + c * wa * uv.x * uv.y;
        float sum = h.x + h.y + h.z;
        vec3 slope = 1.0 + 2.0 * a * w + 3.0 * b * w * w;
        float su = c * uv.y * (wa - uv.x);
        float sv = c * uv.x * (wa - uv.y);
        vec2 sumSlope = vec2(-slope.x + slope.y + 3.0 * su, -slope.x + slope.z + 3.0 * sv);
        vec2 rowB = (vec2(slope.y + su, sv) * sum - h.y * sumSlope) / (sum * sum);
        vec2 rowC = (vec2(su, slope.z + sv) * sum - h.z * sumSlope) / (sum * sum);
        vec2 f = h.yz / sum - target;
        float det = rowB.x * rowC.y - rowB.y * rowC.x;
        uv -= vec2(rowC.y * f.x - rowB.y * f.y, rowB.x * f.y - rowC.x * f.x) / det;
    }
    return uv;
}

vec4 faceTexel(int column, int face) {
    return texelFetch(latticeFaces, ivec2(column, face), 0);
}

int canonicalId(int face, int i, int j) {
    ivec4 corners = ivec4(faceTexel(5, face));
    vec4 at = faceTexel(6, face);
    vec4 by = faceTexel(7, face);
    int cols = latticeCols;
    if (i == 0 && j == 0) return corners.x;
    if (i == 0 && j == cols) return corners.y;
    if (i == cols) return corners.z;
    if (i == 0) return int(at.x) + j * int(by.x);
    if (j == 0) return int(at.y) + i * int(by.y);
    if (j == cols - i) return int(at.z) + i * int(by.z);
    return corners.w + (i - 1) * (cols - 2) - (i - 1) * (i - 2) / 2 + j - 1;
}

// A number in [0, 1] of the id alone, the same on every GPU.
float shadeOf(int id) {
    uint h = uint(id) * 747796405u + 2891336453u;
    h = ((h >> ((h >> 28u) + 4u)) ^ h) * 277803737u;
    return float((h >> 22u) ^ h) / 4294967295.0;
}

// 1 on a cell's hexagonal edge, 0 inside; `offset` is from the cell's lattice point, in lattice steps.
float cellEdge(vec2 offset, float stepArc, float pixelArc) {
    vec2 p = vec2(offset.x + 0.5 * offset.y, 0.8660254 * offset.y);
    float d = max(abs(p.x), max(abs(dot(p, vec2(0.5, 0.8660254))), abs(dot(p, vec2(0.5, -0.8660254)))));
    float pixel = pixelArc / stepArc;
    return (1.0 - smoothstep(0.4 * pixel, 1.6 * pixel, 0.5 - d)) * smoothstep(EDGE_FROM, EDGE_FULL, 1.0 / pixel);
}

vec2 flagUVof(vec3 ground, vec3 centre, vec3 east, vec3 north, vec2 halfSize, vec2 anchor) {
    vec2 offset = vec2(dot(ground, east), dot(ground, north));
    float reach = length(offset);
    // Not acos: near the flag's centre the dot rounds to 1 and the step flickers to nothing.
    float angle = atan(reach, dot(ground, centre));
    vec2 surface = reach > 1e-6 ? offset / reach * angle : vec2(0.0);
    return surface / halfSize * 0.5 + anchor;
}

// A flag that does not lead the ground covers its own footprint there from its middle, as the leader's would with no fit.
vec2 coverOf(vec2 reach, float aspect) {
    float halfV = clamp(max(reach.y, reach.x / aspect), leastFlagRadius, MAX_FLAG_RADIUS);
    return vec2(halfV * aspect, halfV);
}

// A footprint's two texels: its centre and half extent east, then its half extent north, the landmass's footprint (a
// province's alone) and its tiles. `firstRow` 0 for the provinces', FOOTPRINT_ROWS for the landmasses'.
vec4 footprintTexel(int footprint, int firstRow, int texel) {
    ivec2 at = ivec2((footprint % FOOTPRINT_COLUMNS) * 2 + texel, firstRow + footprint / FOOTPRINT_COLUMNS);
    return texelFetch(footprintFrames, at, 0);
}

vec4 paintedFlag(vec3 ground, vec3 acrossStep, vec3 upStep, vec4 own, int ownNumber, int footprint, int firstRow, float under, sampler2D groundData, float groundCount) {
    if (under < 0.5) return vec4(0.0);
    float row = (under + 0.5) / groundCount;
    vec4 region = texture(groundData, vec2(5.0 / 8.0, row));
    vec4 held = texture(groundData, vec2(7.0 / 8.0, row));

    bool split = splitFlags > 0.5;
    vec4 flag = split ? own : region;
    float share = split ? 1.0 : held.r * 0.94;
    if (flag.z <= 0.0 || share <= 0.0) return vec4(0.0);

    // Only the leader's flag is fitted to the ground and drawn large.
    bool leads = all(equal(region, flag));
    vec4 frame = texture(groundData, vec2(1.0 / 8.0, row));
    vec4 axis = texture(groundData, vec2(3.0 / 8.0, row));
    vec3 centre = frame.xyz;
    vec2 reach = axis.xy;
    // Any other flag is framed by where its own tiles are on this ground, so each wears the part of the flag it would
    // in a flag drawn over them; with no footprint (a table full), by the ground's middle and reach.
    if (!leads && footprint > 0) {
        vec4 place = footprintTexel(footprint, firstRow, 0);
        vec4 extent = footprintTexel(footprint, firstRow, 1);
        if (extent.w > 0.0) {
            centre = place.xyz;
            reach = vec2(place.w, extent.x);
        }
    }
    vec2 halfSize = leads ? vec2(frame.w, axis.w) : coverOf(reach, flag.z / flag.w);
    vec2 anchor = leads ? held.ba : vec2(0.5);

    if (dot(ground, centre) <= 0.0) return vec4(0.0);
    vec3 east = normalize(vec3(centre.z, 0.0, -centre.x));
    if (abs(centre.y) > 0.9999) east = vec3(1.0, 0.0, 0.0);
    vec3 north = cross(centre, east);

    vec2 uv = flagUVof(ground, centre, east, north, halfSize, anchor);
    vec2 across = flagUVof(normalize(ground + acrossStep), centre, east, north, halfSize, anchor) - uv;
    vec2 upward = flagUVof(normalize(ground + upStep), centre, east, north, halfSize, anchor) - uv;

    vec4 slotFrame = texelFetch(slotFrames, ivec2(split ? ownNumber : int(held.g + 0.5), 0), 0);
    bool slotted = slotFrame.z > 0.0;
    vec4 flagFrame = slotted ? slotFrame : atlasFrameOf(flag);

    // Explicit gradients: they follow the ground, which runs on across two tiles of different flags.
    vec2 at = frameUVof(flagFrame, uv);
    vec3 rgb = onWhite(slotted
        ? textureGrad(slotTexture, at, across * flagFrame.zw, upward * flagFrame.zw)
        : textureGrad(atlasTexture, at, across * flagFrame.zw, upward * flagFrame.zw));
    return vec4(rgb, share);
}

void main() {
    // The camera is orthographic: every pixel looks down the same axis, onto the tiles' sphere of radius 1.
    vec3 forward = -vec3(viewMatrix[0][2], viewMatrix[1][2], viewMatrix[2][2]);
    float along = dot(vWorld, forward);
    float inside = along * along - dot(vWorld, vWorld) + 1.0;
    vec3 ground = normalize(vWorld + forward * (-along - sqrt(max(inside, 0.0))));
    vec3 acrossStep = dFdx(ground);
    vec3 upStep = dFdy(ground);
    float pixelArc = max(length(acrossStep), length(upStep));
    if (inside < 0.0) discard;

    int face = 0;
    float best = -2.0;
    for (int f = 0; f < FACES; f++) {
        float facing = dot(ground, faceNormals[f].xyz);
        if (facing > best) {
            best = facing;
            face = f;
        }
    }

    vec3 onFace = ground * (faceNormals[face].w / best) - faceTexel(0, face).xyz;
    float cols = float(latticeCols);
    // The gnomonic inverse gives the warped weights; a warped board's own weights are two Newton steps back.
    vec2 weights = vec2(dot(onFace, faceTexel(3, face).xyz), dot(onFace, faceTexel(4, face).xyz));
    if (latticeWarp.w > 0.5) weights = unwarped(weights);
    vec2 onGrid = weights * cols;
    ivec2 cell = nearestOnGrid(onGrid);
    int i = clamp(cell.y, 0, latticeCols);
    int j = clamp(cell.x, 0, latticeCols - i);

    int id = canonicalId(face, i, j);
    uint kind = texelFetch(terrain, ivec2(id % TERRAIN_COLUMNS, id / TERRAIN_COLUMNS), 0).r;
    if (kind >= LAND_KINDS) {
        // The open sea is the earth's own sphere, with its glint.
        if (kind == OPEN_SEA) discard;
        gl_FragColor = vec4(terrainColours[kind], 1.0);
        return;
    }

    vec3 land = terrainColours[kind] * (1.0 + SHADE_JITTER * (2.0 * shadeOf(id) - 1.0));
    land *= 1.0 - CELL_EDGE * cellEdge(onGrid - vec2(cell), length(faceTexel(1, face).xyz) / cols, pixelArc);
    // Zoomed in, or with the painted flags off, the flags are the tiles' alone: the layer is the bare ground.
    if (flagPaint <= 0.0) {
        gl_FragColor = vec4(land, 1.0);
        return;
    }

    int word = id >> 5;
    uvec2 bits = texelFetch(tileCells, ivec2(word % CELL_COLUMNS, word / CELL_COLUMNS), 0).rg;
    uint bit = uint(id & 31);
    if (((bits.x >> bit) & 1u) == 0u) discard;

    int texel = int(bits.y + popcount(bits.x & ((1u << bit) - 1u)));
    uvec2 tile = texelFetch(tileData, ivec2(texel % TILE_COLUMNS, texel / TILE_COLUMNS), 0).rg;
    float province = float(tile.y);
    int at = int(tile.y);
    float landmass = float(texelFetch(provinceLandmass, ivec2(at % PROVINCE_COLUMNS, at / PROVINCE_COLUMNS), 0).r);
    vec4 own = texelFetch(flagRectangles, ivec2(int(tile.x), 0), 0);
    int footprint = int(texelFetch(footprintData, ivec2(texel % TILE_COLUMNS, texel / TILE_COLUMNS), 0).r);
    int landmassFootprint = footprint > 0 ? int(footprintTexel(footprint, 0, 1).y + 0.5) : 0;

    // Each framing painted only while it shows: both for the fade alone, blended with their coverage.
    vec4 colour = vec4(0.0);
    if (provinceShare < 1.0) colour = paintedFlag(ground, acrossStep, upStep, own, int(tile.x), landmassFootprint, FOOTPRINT_ROWS, landmass, landmassData, landmassCount);
    if (provinceShare > 0.0) {
        vec4 divided = paintedFlag(ground, acrossStep, upStep, own, int(tile.x), footprint, 0, province, provinceData, provinceCount);
        vec4 blend = mix(vec4(colour.rgb * colour.a, colour.a), vec4(divided.rgb * divided.a, divided.a), provinceShare);
        colour = blend.a > 0.0 ? vec4(blend.rgb / blend.a, blend.a) : vec4(0.0);
    }

    if (province > highlight.x - 0.5 && province < highlight.y + 0.5) {
        colour = vec4(mix(colour.rgb, vec3(1.0), HIGHLIGHT_LIFT), max(colour.a, HIGHLIGHT_FLOOR));
    }

    // The tile's own centre, so an effect is drawn tile by tile, as on the tiles.
    vec3 tileCentre;
    if (latticeWarp.w > 0.5) {
        vec3 w = warped(vec3(cols - float(i) - float(j), float(j), float(i)) / cols);
        tileCentre = normalize(faceTexel(0, face).xyz + faceTexel(1, face).xyz * w.y + faceTexel(2, face).xyz * w.z);
    } else {
        tileCentre = normalize(faceTexel(0, face).xyz + (faceTexel(1, face).xyz * float(j) + faceTexel(2, face).xyz * float(i)) / cols);
    }
    TileEffects effects = tileEffectsAt(tileCentre, province);
    colour = heated(steeled(scorched(colour, effects.scorch), effects.fortify), effects.glow);

    // Solid while the tiles fade in over it, then the bare ground in the second half of the handover.
    gl_FragColor = vec4(mix(land, colour.rgb, colour.a * min(1.0, flagPaint * 2.0)), 1.0);
}
