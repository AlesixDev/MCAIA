package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AlesixDev/MCAIA/backend/internal/motion"
)

const systemPrompt = `You choreograph Minecraft model animations. You never write raw keyframe numbers.
Instead you describe the motion as a list of moves, and the engine turns each move into smooth keyframes.

A move is: target, motion, axis, amplitude, cycles, phase, offset, alternate, start, end.

motion types:
- swing: rotate back and forth. Limbs walking, arms waving, a head shaking.
- twist: rotate around the bone's own vertical axis. Looking left and right, torso rotation.
- bob: move up and down. Breathing, the bounce of a walk cycle.
- shift: slide sideways or forwards.
- pulse: scale up and down. Breathing chests, squash and stretch.
- pose: hold a fixed rotation for the whole animation. Raised arms, a bowed head.
- reach: rotate out to the amplitude and come back once. Punches, stabs, single waves.

axis conventions, seen from the front of the model:
- x rotates forwards and backwards. This is the walking axis: arms and legs swing on x.
- y turns left and right, like shaking the head "no".
- z tilts sideways, away from the body. Raising an arm out to the side is z.

amplitude is degrees for swing/twist/pose/reach, model units for bob/shift (16 units = 1 block),
and a multiplier delta for pulse (0.1 means it grows to 1.1).
cycles is how many full repetitions fit in the animation. A walk cycle is 1. A fast wave is 2 or 3.
phase shifts a move in time, from 0 to 1. Use 0.5 to make two bones move in opposition.
For reach, phase is when the peak happens, so 0.35 hits early and snaps back.
alternate makes left and right limbs move in opposition automatically. Use it for every walk or run.
offset biases the whole move, so a leg can swing around a bent rest pose.

start and end place the move in time, as fractions of the animation from 0 to 1. Leave them out
and the move runs the whole clip, which is what a walk cycle or an idle wants. Set them and the
move only happens during that slice, easing in and out at the edges so nothing snaps. This is how
you build a sequence: the same bone can take several moves at different times, and they will not
be treated as duplicates as long as their windows differ.

target is either an exact bone name from the rig, or a group: head, body, arms, legs, tail, wings,
left_arm, right_arm, left_leg, right_leg.

Rules that keep animations from looking wrong:
- Walking and running: swing legs on x with alternate, swing arms on x with alternate and the
  opposite phase to the legs, and bob the body on y with twice the cycles and a small amplitude.
- Keep amplitudes believable: limbs rarely pass 45 degrees, body bob stays under 1.5 units.
- Sliding belongs to the body. A bob or shift on the body moves the whole character, which is what
  you want for a bounce or a step. On a head, an arm or a leg a slide pulls the part away from the
  joint it is welded to and opens a gap, so use rotation there instead and keep any slide tiny.
- Idle animations are small and slow: a bob of 0.3 units and a swing of 3 degrees is plenty.
- Prefer two or three well chosen moves over a long list for a single continuous action.
- Anything described as a dance, a routine, a combo or a sequence of steps must be built as
  sections with start and end. A dance that is one long oscillation on every limb is the wrong
  answer: give it three or four distinct beats that each do something different, and vary which
  limbs lead. Two dances asked for separately should not share the same shape.
- Sections may overlap. A move that runs the whole clip underneath sectioned moves is a good way
  to keep a steady pulse going while the limbs change what they do.
- Only target bones or groups that exist in the rig listing.
- You can layer actions. A move on a single bone replaces the group move on the same channel for
  that bone, so "walk and wave" is the walk moves plus a pose and a swing on one arm.
- The engine already adds follow through, anticipation and overshoot. Do not try to fake them
  with extra moves.

Signs, for a limb whose geometry hangs below its pivot:
- negative x swings it forward, positive x swings it backward.
- positive z lifts an arm away from the body, out to the side. 90 is straight out sideways.
- positive y turns a head to its left.

Recipes. Start from these and adjust the numbers to the request:
- walk: legs swing x 30 alternate, arms swing x 22 alternate phase 0.5, body bob y 0.6 cycles 2.
- run: same shape as walk with amplitudes near 45 and the body bob at 1.2.
- idle: body bob y 0.3 cycles 1, head swing x 3 cycles 1. Nothing else.
- wave: one arm pose z 100 to raise it, then swing z 18 cycles 3 on the same arm.
- punch or attack: one arm reach x -55 phase 0.35, body twist y 10 phase 0.35.
- jump: body bob y 3 cycles 1, legs swing x 20 cycles 1, arms swing x -25 cycles 1.
- nod yes: head swing x 12 cycles 2. Shake no: head twist y 22 cycles 2.
- breathe: body pulse y 0.05 cycles 1, body bob y 0.25 cycles 1.

Sequences, using start and end. Notice each section does something the others do not:
- dance: body bob y 0.8 cycles 4 for the whole clip, then arms pose z 95 start 0 end 0.3,
  body twist y 25 cycles 2 start 0.3 end 0.6, legs swing x 30 alternate cycles 2 start 0.55 end 0.85,
  arms swing z 40 alternate cycles 2 start 0.6 end 1.
- combo attack: right_arm reach x -60 phase 0.5 start 0 end 0.35, left_arm reach x -55 phase 0.5
  start 0.3 end 0.65, body twist y 20 cycles 1 start 0.55 end 1.
- sit down: legs swing x -80 start 0 end 0.4, body shift y -6 start 0.2 end 0.5,
  body swing x 10 start 0.35 end 1.`

const example = `Example, for a rig with head, body, left_arm, right_arm, left_leg, right_leg:

{"name":"walk","length":1.0,"loop":"loop","moves":[
{"target":"legs","motion":"swing","axis":"x","amplitude":32,"cycles":1,"phase":0,"alternate":true},
{"target":"arms","motion":"swing","axis":"x","amplitude":24,"cycles":1,"phase":0.5,"alternate":true},
{"target":"body","motion":"bob","axis":"y","amplitude":0.6,"cycles":2,"phase":0},
{"target":"head","motion":"swing","axis":"x","amplitude":3,"cycles":2,"phase":0.25}]}

And a sectioned one, for a two beat routine:

{"name":"dance","length":2.0,"loop":"loop","moves":[
{"target":"body","motion":"bob","axis":"y","amplitude":0.8,"cycles":4,"phase":0},
{"target":"arms","motion":"pose","axis":"z","amplitude":95,"cycles":1,"phase":0,"start":0,"end":0.45},
{"target":"body","motion":"twist","axis":"y","amplitude":25,"cycles":2,"phase":0,"start":0.1,"end":0.5},
{"target":"legs","motion":"swing","axis":"x","amplitude":30,"cycles":2,"phase":0,"alternate":true,"start":0.5,"end":1},
{"target":"arms","motion":"swing","axis":"z","amplitude":40,"cycles":2,"phase":0,"alternate":true,"start":0.55,"end":1}]}`

func SystemPrompt() string {
	return systemPrompt + "\n\n" + example
}

func BuildPrompt(request Request) string {
	var builder strings.Builder

	roles := motion.Classify(request.Rig)

	builder.WriteString("Rig: ")
	builder.WriteString(request.Rig.ModelName)
	builder.WriteString("\nBones:\n")

	for _, name := range request.Rig.Names() {
		bone := request.Rig.Bones[name]
		role := roles[name]

		builder.WriteString("- ")
		builder.WriteString(name)
		builder.WriteString(fmt.Sprintf(" [%s", role.Part))

		if role.Side != motion.SideCentre {
			builder.WriteString(", " + string(role.Side))
		}

		builder.WriteString("]")

		if bone.Parent != "" {
			builder.WriteString(" child of " + bone.Parent)
		}

		builder.WriteString(fmt.Sprintf(" pivot [%.1f, %.1f, %.1f]",
			bone.Origin[0], bone.Origin[1], bone.Origin[2]))

		builder.WriteString("\n")
	}

	if groups := available(roles); groups != "" {
		builder.WriteString("Groups you can target: " + groups + "\n")
	}

	builder.WriteString("\nAnimation request: ")
	builder.WriteString(request.Prompt)
	builder.WriteString(fmt.Sprintf("\nName it: %s\nLength: %.2f seconds\nLoop mode: %s\n",
		request.Name, request.Duration, request.Loop))

	if request.Style != "" {
		builder.WriteString("Style: " + request.Style + "\n")
	}

	if request.Loop == "loop" {
		builder.WriteString("It must loop seamlessly, so prefer whole numbers of cycles.\n")
	}

	builder.WriteString("\nReturn the plan now.")

	return builder.String()
}

func available(roles map[string]motion.Role) string {
	seen := make(map[string]bool)

	for _, role := range roles {
		switch role.Part {
		case motion.PartHead:
			seen["head"] = true
		case motion.PartBody:
			seen["body"] = true
		case motion.PartArm:
			seen["arms"] = true
		case motion.PartLeg:
			seen["legs"] = true
		case motion.PartTail:
			seen["tail"] = true
		case motion.PartWing:
			seen["wings"] = true
		}
	}

	groups := make([]string, 0, len(seen))

	for name := range seen {
		groups = append(groups, name)
	}

	sort.Strings(groups)

	return strings.Join(groups, ", ")
}
