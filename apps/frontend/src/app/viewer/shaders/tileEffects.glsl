// Must match MAX_BLASTS and BLAST_TIMELINE in domain/blast.ts.
#define MAX_BLASTS 4
const float BLAST_FALL = 0.8;
const float BLAST_SCORCH = 5.0;

uniform float time;
uniform vec4 blasts[MAX_BLASTS];
uniform float blastRadii[MAX_BLASTS];
uniform float motion;

// Must match MAX_FORTIFY_WAVES, FORTIFY_WAVE_SECONDS and FORTIFY_GLOW_SECONDS in domain/fortifyWave.ts.
#define MAX_FORTIFIES 4
const float FORTIFY_WAVE = 1.2;
const float FORTIFY_GLOW = 0.7;

uniform vec4 fortifies[MAX_FORTIFIES];
uniform float fortifyLandmass[MAX_FORTIFIES];
uniform float fortifyReach[MAX_FORTIFIES];

struct TileEffects {
    float glow;
    float scorch;
    float fortify;
    float swell;
    vec3 push;
};

TileEffects tileEffectsAt(vec3 ground, float landmass) {
    TileEffects effects = TileEffects(0.0, 0.0, 0.0, 0.0, vec3(0.0));

    for (int i = 0; i < MAX_BLASTS; i++) {
        float radius = blastRadii[i];
        if (radius <= 0.0) continue;

        float t = time - blasts[i].w;
        if (t < 0.0 || t > BLAST_FALL + BLAST_SCORCH) continue;

        float along = dot(ground, blasts[i].xyz);
        if (along < cos(min(radius * 3.0, 3.14159))) continue;

        float d = acos(min(along, 1.0)) / radius;

        if (t < BLAST_FALL) continue;

        float s = t - BLAST_FALL;

        float front = 2.6 * (1.0 - exp(-s * 3.2));
        float ring = exp(-pow((d - front) * 2.8, 2.0)) * exp(-s * 1.6);
        float flash = (1.0 - smoothstep(0.0, 1.1, d)) * exp(-s * 5.0);
        effects.glow = max(effects.glow, max(ring, flash));

        vec3 outward = ground - blasts[i].xyz * along;
        float reach = length(outward);
        if (reach > 1e-5) effects.push += motion * (outward / reach) * ring * radius * 0.3;
        effects.push += motion * ground * flash * radius * 0.25;
        effects.swell = max(effects.swell, ring * 1.2 + flash * 1.8);

        float crater = 1.0 - smoothstep(0.85, 1.1, d);
        effects.scorch = max(effects.scorch, crater * (1.0 - smoothstep(0.0, BLAST_SCORCH, s)));
    }

    for (int i = 0; i < MAX_FORTIFIES; i++) {
        if (abs(landmass - fortifyLandmass[i]) > 0.5) continue;

        float t = time - fortifies[i].w;
        if (t < 0.0 || t > FORTIFY_WAVE + FORTIFY_GLOW) continue;

        float d = acos(clamp(dot(ground, fortifies[i].xyz), -1.0, 1.0));
        float arrival = fortifyReach[i] > 0.0 ? d / fortifyReach[i] * FORTIFY_WAVE : 0.0;
        float since = t - arrival;
        if (since < 0.0) continue;

        float lit = exp(-since * 5.0);
        effects.fortify = max(effects.fortify, lit);
        effects.swell = max(effects.swell, lit * 0.6);
    }

    return effects;
}
