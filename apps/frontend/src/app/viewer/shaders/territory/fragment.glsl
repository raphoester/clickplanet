uniform sampler2D landmassData;
uniform float landmassCount;

uniform float flagPaint;
// 1: each tile paints its own holder's flag. 0: the landmass's leader paints all of it, at its share.
uniform float splitFlags;

uniform float highlight;

uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform sampler2D slotTexture;
uniform sampler2D flagRectangles;

// Must match SLOT_COUNT in slotRanking.ts.
#define FLAG_SLOTS 8
uniform vec4 slotFrames[FLAG_SLOTS];
uniform vec4 slotRegions[FLAG_SLOTS];

// Must match latticeCells.ts, which finds a direction's cell the same way.
#define FACES 20
uniform vec4 faceNormals[FACES];
uniform sampler2D latticeFaces;
uniform int latticeCols;
uniform highp usampler2D tileCells;
uniform highp usampler2D tileData;

// Must match CELL_COLUMNS in territoryLayer.ts and TILE_COLUMNS in tileField.ts.
const int CELL_COLUMNS = 256;
const int TILE_COLUMNS = 512;

#include ../flagFrame.glsl;
#include ../tileEffects.glsl;
#include ../effectColours.glsl;

varying vec3 vWorld;

const float HIGHLIGHT_LIFT = 0.2;
const float HIGHLIGHT_FLOOR = 0.35;

// Must match MAX_FLAG_RADIUS in borderField.ts.
const float MAX_FLAG_RADIUS = 0.42;
const float LEAST_FLAG_RADIUS = 0.004;

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

vec2 flagUVof(vec3 ground, vec3 centre, vec3 east, vec3 north, vec2 halfSize, vec2 anchor) {
    vec2 offset = vec2(dot(ground, east), dot(ground, north));
    float reach = length(offset);
    // Not acos: near the flag's centre the dot rounds to 1 and the step flickers to nothing.
    float angle = atan(reach, dot(ground, centre));
    vec2 surface = reach > 1e-6 ? offset / reach * angle : vec2(0.0);
    return surface / halfSize * 0.5 + anchor;
}

// A flag that does not lead the landmass covers its ground from the middle, as the leader's would with no fit.
vec2 coverOf(vec2 reach, float aspect) {
    float halfV = clamp(max(reach.y, reach.x / aspect), LEAST_FLAG_RADIUS, MAX_FLAG_RADIUS);
    return vec2(halfV * aspect, halfV);
}

vec4 paintedFlag(vec3 ground, vec3 acrossStep, vec3 upStep, vec4 own, float landmass) {
    float row = (landmass + 0.5) / landmassCount;
    vec4 region = texture(landmassData, vec2(5.0 / 8.0, row));
    vec4 held = texture(landmassData, vec2(7.0 / 8.0, row));

    bool split = splitFlags > 0.5;
    vec4 flag = split ? own : region;
    float share = split ? 1.0 : held.r * 0.94;
    if (flag.z <= 0.0 || share <= 0.0) return vec4(0.0);

    // Only the leader's flag is fitted to the ground and drawn large.
    bool leads = all(equal(region, flag));
    vec4 frame = texture(landmassData, vec2(1.0 / 8.0, row));
    vec4 axis = texture(landmassData, vec2(3.0 / 8.0, row));
    vec2 halfSize = leads ? vec2(frame.w, axis.w) : coverOf(axis.xy, flag.z / flag.w);
    vec2 anchor = leads ? held.ba : vec2(0.5);

    vec3 centre = frame.xyz;
    if (dot(ground, centre) <= 0.0) return vec4(0.0);
    vec3 east = normalize(vec3(centre.z, 0.0, -centre.x));
    if (abs(centre.y) > 0.9999) east = vec3(1.0, 0.0, 0.0);
    vec3 north = cross(centre, east);

    vec2 uv = flagUVof(ground, centre, east, north, halfSize, anchor);
    vec2 across = flagUVof(normalize(ground + acrossStep), centre, east, north, halfSize, anchor) - uv;
    vec2 upward = flagUVof(normalize(ground + upStep), centre, east, north, halfSize, anchor) - uv;

    vec4 slotFrame = vec4(0.0);
    for (int i = 0; i < FLAG_SLOTS; i++) {
        if (all(equal(slotRegions[i], flag))) slotFrame = slotFrames[i];
    }
    bool slotted = slotFrame.z > 0.0;
    vec4 flagFrame = slotted ? slotFrame : atlasFrameOf(flag);

    // Explicit gradients: they follow the ground, which runs on across two tiles of different flags.
    vec2 at = frameUVof(flagFrame, uv);
    vec3 rgb = slotted
        ? textureGrad(slotTexture, at, across * flagFrame.zw, upward * flagFrame.zw).rgb
        : textureGrad(atlasTexture, at, across * flagFrame.zw, upward * flagFrame.zw).rgb;
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
    ivec2 cell = nearestOnGrid(vec2(dot(onFace, faceTexel(3, face).xyz), dot(onFace, faceTexel(4, face).xyz)) * cols);
    int i = clamp(cell.y, 0, latticeCols);
    int j = clamp(cell.x, 0, latticeCols - i);

    int id = canonicalId(face, i, j);
    int word = id >> 5;
    uvec2 bits = texelFetch(tileCells, ivec2(word % CELL_COLUMNS, word / CELL_COLUMNS), 0).rg;
    uint bit = uint(id & 31);
    if (((bits.x >> bit) & 1u) == 0u) discard;

    int texel = int(bits.y + popcount(bits.x & ((1u << bit) - 1u)));
    uvec2 tile = texelFetch(tileData, ivec2(texel % TILE_COLUMNS, texel / TILE_COLUMNS), 0).rg;
    float landmass = float(tile.y);
    vec4 own = texelFetch(flagRectangles, ivec2(int(tile.x), 0), 0);

    vec4 colour = landmass > 0.5 ? paintedFlag(ground, acrossStep, upStep, own, landmass) : vec4(0.0);

    if (highlight > 0.5 && abs(landmass - highlight) < 0.5) {
        colour = vec4(mix(colour.rgb, vec3(1.0), HIGHLIGHT_LIFT), max(colour.a, HIGHLIGHT_FLOOR));
    }

    // The tile's own centre, so an effect is drawn tile by tile, as on the tiles.
    vec3 tileCentre = normalize(faceTexel(0, face).xyz + (faceTexel(1, face).xyz * float(j) + faceTexel(2, face).xyz * float(i)) / cols);
    TileEffects effects = tileEffectsAt(tileCentre, landmass);
    colour = heated(steeled(scorched(colour, effects.scorch), effects.fortify), effects.glow);

    // Solid while the tiles fade in over it, then gone in the second half of the handover.
    gl_FragColor = vec4(colour.rgb, colour.a * min(1.0, flagPaint * 2.0));
}
