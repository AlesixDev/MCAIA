import { Keyframe, Vec3 } from "@/lib/api";

function lerp(from: number, to: number, ratio: number) {
  return from + (to - from) * ratio;
}

function spline(p0: number, p1: number, p2: number, p3: number, ratio: number) {
  const squared = ratio * ratio;
  const cubed = squared * ratio;

  return (
    0.5 *
    (2 * p1 +
      (p2 - p0) * ratio +
      (2 * p0 - 5 * p1 + 4 * p2 - p3) * squared +
      (3 * p1 - p0 - 3 * p2 + p3) * cubed)
  );
}

export function sample(frames: Keyframe[] | undefined, time: number, fallback: number): Vec3 {
  if (!frames || frames.length === 0) {
    return [fallback, fallback, fallback];
  }

  if (time <= frames[0].time) {
    return frames[0].value;
  }

  const last = frames[frames.length - 1];

  if (time >= last.time) {
    return last.value;
  }

  let index = 0;

  while (index < frames.length - 1 && frames[index + 1].time <= time) {
    index += 1;
  }

  const from = frames[index];
  const to = frames[index + 1];

  if (from.interpolation === "step") {
    return from.value;
  }

  const span = to.time - from.time;
  const ratio = span <= 0 ? 0 : (time - from.time) / span;

  if (from.interpolation !== "catmullrom") {
    return [
      lerp(from.value[0], to.value[0], ratio),
      lerp(from.value[1], to.value[1], ratio),
      lerp(from.value[2], to.value[2], ratio),
    ];
  }

  const before = frames[Math.max(index - 1, 0)];
  const after = frames[Math.min(index + 2, frames.length - 1)];

  return [
    spline(before.value[0], from.value[0], to.value[0], after.value[0], ratio),
    spline(before.value[1], from.value[1], to.value[1], after.value[1], ratio),
    spline(before.value[2], from.value[2], to.value[2], after.value[2], ratio),
  ];
}
