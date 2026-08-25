package animation

type Vec3 [3]float64

type Interpolation string

const (
	InterpolationLinear     Interpolation = "linear"
	InterpolationCatmullRom Interpolation = "catmullrom"
	InterpolationStep       Interpolation = "step"
)

type LoopMode string

const (
	LoopNone  LoopMode = "once"
	LoopCycle LoopMode = "loop"
	LoopHold  LoopMode = "hold_on_last_frame"
)

type Channel string

const (
	ChannelRotation Channel = "rotation"
	ChannelPosition Channel = "position"
	ChannelScale    Channel = "scale"
)

type Keyframe struct {
	Time          float64       `json:"time"`
	Value         Vec3          `json:"value"`
	Interpolation Interpolation `json:"interpolation,omitempty"`
}

type Track struct {
	Rotation []Keyframe `json:"rotation,omitempty"`
	Position []Keyframe `json:"position,omitempty"`
	Scale    []Keyframe `json:"scale,omitempty"`
}

type Animation struct {
	Name        string           `json:"name"`
	Length      float64          `json:"length"`
	Loop        LoopMode         `json:"loop"`
	Description string           `json:"description,omitempty"`
	Bones       map[string]Track `json:"bones"`
}

func (t *Track) Channel(channel Channel) []Keyframe {
	switch channel {
	case ChannelRotation:
		return t.Rotation
	case ChannelPosition:
		return t.Position
	case ChannelScale:
		return t.Scale
	}

	return nil
}

func (t *Track) SetChannel(channel Channel, frames []Keyframe) {
	switch channel {
	case ChannelRotation:
		t.Rotation = frames
	case ChannelPosition:
		t.Position = frames
	case ChannelScale:
		t.Scale = frames
	}
}

func (t Track) IsEmpty() bool {
	return len(t.Rotation) == 0 && len(t.Position) == 0 && len(t.Scale) == 0
}

func Channels() []Channel {
	return []Channel{ChannelRotation, ChannelPosition, ChannelScale}
}
