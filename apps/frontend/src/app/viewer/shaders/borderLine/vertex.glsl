uniform vec2 halfViewport;

uniform float halfWidth;

uniform float lift;

uniform float limb;

attribute vec2 corner;
attribute vec3 from;
attribute vec3 to;

out float vAcross;

void main() {
    vec3 headDirection = normalize(from);
    vec3 tailDirection = normalize(to);

    float slack = limb + (halfWidth + 1.0) / max(halfViewport.y * projectionMatrix[1][1], 1.0);
    if ((normalMatrix * headDirection).z < -slack && (normalMatrix * tailDirection).z < -slack) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }

    vec4 head = projectionMatrix * modelViewMatrix * vec4(headDirection * lift, 1.0);
    vec4 tail = projectionMatrix * modelViewMatrix * vec4(tailDirection * lift, 1.0);

    vec2 span = tail.xy / tail.w * halfViewport - head.xy / head.w * halfViewport;
    float length2 = length(span);
    vec2 along = length2 > 1e-6 ? span / length2 : vec2(1.0, 0.0);

    vec4 clip = corner.x < 0.0 ? head : tail;
    vec2 offset = vec2(-along.y, along.x) * corner.y * halfWidth + along * corner.x * halfWidth;
    clip.xy += offset / halfViewport * clip.w;

    vAcross = corner.y * halfWidth;
    gl_Position = clip;
}
