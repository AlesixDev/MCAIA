"use client";

import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { Pause, Play, RotateCcw } from "lucide-react";

import { Animation, Cube, Rig } from "@/lib/api";
import { sample } from "@/lib/sample";
import { cn } from "@/lib/utils";

const degree = Math.PI / 180;

const sides = ["east", "west", "up", "down", "south", "north"] as const;

type Rigged = {
  root: THREE.Group;
  bones: Map<string, THREE.Group>;
  radius: number;
  centre: THREE.Vector3;
  floor: number;
};

type Rest = {
  position: THREE.Vector3;
  rotation: THREE.Euler;
};

function readColor(name: string, fallback: string) {
  if (typeof window === "undefined") {
    return fallback;
  }

  const raw = getComputedStyle(document.documentElement).getPropertyValue(name).trim();

  return raw ? `rgb(${raw.split(/\s+/).join(",")})` : fallback;
}

function loadTextures(rig: Rig) {
  const loader = new THREE.TextureLoader();

  return (rig.textures ?? [])
    .filter((entry) => entry.source.startsWith("data:"))
    .map((entry) => {
      const map = loader.load(entry.source);

      map.colorSpace = THREE.SRGBColorSpace;
      map.magFilter = THREE.NearestFilter;
      map.minFilter = THREE.NearestFilter;
      map.generateMipmaps = false;

      return new THREE.MeshLambertMaterial({
        map,
        transparent: true,
        alphaTest: 0.05,
        side: THREE.DoubleSide,
      });
    });
}

function applyUV(geometry: THREE.BoxGeometry, cube: Cube, resolution: [number, number]) {
  const attribute = geometry.getAttribute("uv");
  const [width, height] = resolution;

  sides.forEach((side, index) => {
    const face = cube.faces?.[side];

    if (!face) {
      return;
    }

    const [x1, y1, x2, y2] = face.uv;

    let corners = [
      [x1 / width, 1 - y1 / height],
      [x2 / width, 1 - y1 / height],
      [x1 / width, 1 - y2 / height],
      [x2 / width, 1 - y2 / height],
    ];

    const turns = Math.round((face.rotation ?? 0) / 90) % 4;

    for (let turn = 0; turn < turns; turn += 1) {
      corners = [corners[2], corners[0], corners[3], corners[1]];
    }

    corners.forEach((corner, slot) => attribute.setXY(index * 4 + slot, corner[0], corner[1]));
  });

  attribute.needsUpdate = true;
}

function cubeMaterials(cube: Cube, textures: THREE.Material[], fallback: THREE.Material) {
  return sides.map((side) => {
    const face = cube.faces?.[side];

    if (!face) {
      return fallback;
    }

    return textures[face.texture] ?? fallback;
  });
}

function buildRig(
  rig: Rig,
  animated: Set<string>,
  textures: THREE.Material[],
  outline: THREE.LineBasicMaterial | null,
): Rigged {
  const root = new THREE.Group();
  const bones = new Map<string, THREE.Group>();
  const box = new THREE.Box3();
  const resolution = rig.resolution ?? [16, 16];

  const plain = new THREE.MeshLambertMaterial({ color: readColor("--line-strong", "#3a3a3a") });
  const active = new THREE.MeshLambertMaterial({ color: readColor("--accent", "#3c83f6") });

  const walk = (name: string, parent: THREE.Object3D, parentOrigin: THREE.Vector3) => {
    const bone = rig.bones[name];

    if (!bone) {
      return;
    }

    const origin = new THREE.Vector3(bone.origin[0], bone.origin[1], bone.origin[2]);
    const pivot = new THREE.Group();

    pivot.position.copy(origin).sub(parentOrigin);
    pivot.rotation.order = "ZYX";
    pivot.rotation.set(
      bone.rotation[0] * degree,
      bone.rotation[1] * degree,
      bone.rotation[2] * degree,
    );

    pivot.userData.rest = {
      position: pivot.position.clone(),
      rotation: pivot.rotation.clone(),
    } satisfies Rest;

    parent.add(pivot);
    bones.set(name, pivot);

    const fallback = animated.has(name) ? active : plain;

    for (const cube of bone.cubes ?? []) {
      const inflate = cube.inflate ?? 0;
      const from = new THREE.Vector3(cube.from[0], cube.from[1], cube.from[2]).subScalar(inflate);
      const to = new THREE.Vector3(cube.to[0], cube.to[1], cube.to[2]).addScalar(inflate);

      const size = new THREE.Vector3(
        Math.max(Math.abs(to.x - from.x), 0.01),
        Math.max(Math.abs(to.y - from.y), 0.01),
        Math.max(Math.abs(to.z - from.z), 0.01),
      );

      const geometry = new THREE.BoxGeometry(size.x, size.y, size.z);

      applyUV(geometry, cube, resolution);

      const mesh = new THREE.Mesh(geometry, cubeMaterials(cube, textures, fallback));

      const anchor = new THREE.Group();

      anchor.position.set(
        cube.origin[0] - origin.x,
        cube.origin[1] - origin.y,
        cube.origin[2] - origin.z,
      );

      anchor.rotation.order = "ZYX";
      anchor.rotation.set(
        cube.rotation[0] * degree,
        cube.rotation[1] * degree,
        cube.rotation[2] * degree,
      );

      mesh.position.set(
        (from.x + to.x) / 2 - cube.origin[0],
        (from.y + to.y) / 2 - cube.origin[1],
        (from.z + to.z) / 2 - cube.origin[2],
      );

      anchor.add(mesh);

      if (outline) {
        const edges = new THREE.LineSegments(new THREE.EdgesGeometry(geometry), outline);

        edges.position.copy(mesh.position);
        anchor.add(edges);
      }

      pivot.add(anchor);

      box.expandByPoint(from);
      box.expandByPoint(to);
    }

    for (const child of bone.children ?? []) {
      walk(child, pivot, origin);
    }
  };

  for (const name of rig.roots) {
    walk(name, root, new THREE.Vector3());
  }

  if (box.isEmpty()) {
    box.setFromCenterAndSize(new THREE.Vector3(0, 8, 0), new THREE.Vector3(16, 16, 16));
  }

  const centre = box.getCenter(new THREE.Vector3());
  const radius = Math.max(box.getSize(new THREE.Vector3()).length() / 2, 4);

  return { root, bones, radius, centre, floor: box.min.y };
}

export function ModelPreview({
  rig,
  animation,
  className,
}: {
  rig: Rig;
  animation: Animation;
  className?: string;
}) {
  const mount = useRef<HTMLDivElement>(null);
  const [playing, setPlaying] = useState(true);
  const [time, setTime] = useState(0);

  const playingRef = useRef(playing);
  const timeRef = useRef(0);
  const seekRef = useRef<number | null>(null);
  const resetRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    playingRef.current = playing;
  }, [playing]);

  useEffect(() => {
    const container = mount.current;

    if (!container) {
      return;
    }

    const animatedBones = new Set(Object.keys(animation.bones));
    const scene = new THREE.Scene();
    const textures = loadTextures(rig);

    const outline = textures.length
      ? null
      : new THREE.LineBasicMaterial({
          color: readColor("--body", "#f0f0f0"),
          transparent: true,
          opacity: 0.14,
        });

    const rigged = buildRig(rig, animatedBones, textures, outline);

    scene.add(rigged.root);

    const gridColor = readColor("--line-strong", "#3a3a3a");
    const grid = new THREE.GridHelper(rigged.radius * 4, 12, gridColor, gridColor);

    grid.position.set(rigged.centre.x, rigged.floor, rigged.centre.z);
    grid.material.transparent = true;
    grid.material.opacity = 0.22;
    scene.add(grid);

    scene.add(new THREE.AmbientLight(0xffffff, 0.85));

    const key = new THREE.DirectionalLight(0xffffff, 0.35);
    key.position.set(1, 1.4, 1);
    scene.add(key);

    const fill = new THREE.DirectionalLight(0xffffff, 0.15);
    fill.position.set(-1, 0.4, -0.8);
    scene.add(fill);

    const camera = new THREE.PerspectiveCamera(38, 1, 0.1, 1000);
    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });

    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    container.appendChild(renderer.domElement);

    const orbit = { yaw: Math.PI * 0.25, pitch: 0.28, distance: rigged.radius * 3.4 };
    const home = { ...orbit };

    const place = () => {
      camera.position.set(
        rigged.centre.x + orbit.distance * Math.cos(orbit.pitch) * Math.sin(orbit.yaw),
        rigged.centre.y + orbit.distance * Math.sin(orbit.pitch),
        rigged.centre.z + orbit.distance * Math.cos(orbit.pitch) * Math.cos(orbit.yaw),
      );

      camera.lookAt(rigged.centre);
    };

    resetRef.current = () => {
      orbit.yaw = home.yaw;
      orbit.pitch = home.pitch;
      orbit.distance = home.distance;
      place();
    };

    const resize = () => {
      const { clientWidth, clientHeight } = container;

      if (clientWidth === 0 || clientHeight === 0) {
        return;
      }

      camera.aspect = clientWidth / clientHeight;
      camera.updateProjectionMatrix();
      renderer.setSize(clientWidth, clientHeight, false);
    };

    const observer = new ResizeObserver(resize);
    observer.observe(container);
    resize();
    place();

    let dragging = false;
    let lastX = 0;
    let lastY = 0;

    const down = (event: PointerEvent) => {
      dragging = true;
      lastX = event.clientX;
      lastY = event.clientY;
      container.setPointerCapture(event.pointerId);
    };

    const move = (event: PointerEvent) => {
      if (!dragging) {
        return;
      }

      orbit.yaw -= (event.clientX - lastX) * 0.01;
      orbit.pitch = Math.max(-1.2, Math.min(1.2, orbit.pitch + (event.clientY - lastY) * 0.008));
      lastX = event.clientX;
      lastY = event.clientY;
      place();
    };

    const up = (event: PointerEvent) => {
      dragging = false;
      container.releasePointerCapture(event.pointerId);
    };

    const wheel = (event: WheelEvent) => {
      event.preventDefault();
      orbit.distance = Math.max(
        rigged.radius * 1.4,
        Math.min(rigged.radius * 9, orbit.distance + event.deltaY * 0.05),
      );
      place();
    };

    container.addEventListener("pointerdown", down);
    container.addEventListener("pointermove", move);
    container.addEventListener("pointerup", up);
    container.addEventListener("pointercancel", up);
    container.addEventListener("wheel", wheel, { passive: false });

    const apply = (at: number) => {
      for (const [name, pivot] of rigged.bones) {
        const track = animation.bones[name];
        const rest = pivot.userData.rest as Rest;

        if (!track) {
          pivot.rotation.copy(rest.rotation);
          pivot.position.copy(rest.position);
          pivot.scale.set(1, 1, 1);

          continue;
        }

        const rotation = sample(track.rotation, at, 0);
        const position = sample(track.position, at, 0);
        const scale = sample(track.scale, at, 1);

        pivot.rotation.set(
          rest.rotation.x - rotation[0] * degree,
          rest.rotation.y - rotation[1] * degree,
          rest.rotation.z + rotation[2] * degree,
        );

        pivot.position.set(
          rest.position.x - position[0],
          rest.position.y + position[1],
          rest.position.z + position[2],
        );

        pivot.scale.set(scale[0], scale[1], scale[2]);
      }
    };

    const length = Math.max(animation.length, 0.001);

    let frame = 0;
    let previous = performance.now();

    const tick = (now: number) => {
      const delta = (now - previous) / 1000;
      previous = now;

      if (seekRef.current !== null) {
        timeRef.current = seekRef.current;
        seekRef.current = null;
      } else if (playingRef.current) {
        const next = timeRef.current + delta;

        if (next < length) {
          timeRef.current = next;
        } else if (animation.loop === "loop") {
          timeRef.current = next % length;
        } else {
          timeRef.current = length;
          playingRef.current = false;
          setPlaying(false);
        }
      }

      apply(timeRef.current);
      setTime(timeRef.current);
      renderer.render(scene, camera);

      frame = requestAnimationFrame(tick);
    };

    frame = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();

      container.removeEventListener("pointerdown", down);
      container.removeEventListener("pointermove", move);
      container.removeEventListener("pointerup", up);
      container.removeEventListener("pointercancel", up);
      container.removeEventListener("wheel", wheel);

      renderer.dispose();
      container.removeChild(renderer.domElement);

      for (const material of textures) {
        material.map?.dispose();
        material.dispose();
      }

      grid.geometry.dispose();
      grid.material.dispose();
      outline?.dispose();

      scene.traverse((object) => {
        if (object instanceof THREE.Mesh || object instanceof THREE.LineSegments) {
          object.geometry.dispose();
        }
      });
    };
  }, [rig, animation]);

  function restart() {
    seekRef.current = 0;
    setPlaying(true);
  }

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div
        ref={mount}
        className="h-[220px] w-full cursor-grab overflow-hidden rounded-lg bg-surface-sunken active:cursor-grabbing"
      />

      <div className="flex items-center gap-2.5">
        <button
          onClick={() => (time >= animation.length ? restart() : setPlaying((state) => !state))}
          aria-label={playing ? "Pause" : "Play"}
          className="flex size-7 shrink-0 items-center justify-center rounded-lg border border-line text-muted transition-colors hover:border-line-strong hover:text-body"
        >
          {playing ? <Pause size={12} /> : <Play size={12} />}
        </button>

        <input
          type="range"
          min={0}
          max={animation.length}
          step={0.01}
          value={time}
          onChange={(event) => {
            seekRef.current = Number(event.target.value);
            setPlaying(false);
          }}
          className="h-1 min-w-0 flex-1 cursor-pointer appearance-none rounded-full bg-line accent-[rgb(var(--accent))]"
        />

        <span className="w-14 shrink-0 text-right font-mono text-[10.5px] text-muted">
          {time.toFixed(2)}s
        </span>

        <button
          onClick={() => resetRef.current?.()}
          aria-label="Reset view"
          className="flex size-7 shrink-0 items-center justify-center rounded-lg border border-line text-muted transition-colors hover:border-line-strong hover:text-body"
        >
          <RotateCcw size={12} />
        </button>
      </div>
    </div>
  );
}
