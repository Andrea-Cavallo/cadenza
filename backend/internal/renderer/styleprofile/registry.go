package styleprofile

import "fmt"

// Registry is the source of truth for available style profiles, keyed by name.
type Registry struct {
	profiles map[string]*StyleProfile
}

func NewRegistry() *Registry {
	r := &Registry{profiles: make(map[string]*StyleProfile)}
	r.Register(&BassProgressive)
	r.Register(&BassDriving)
	r.Register(&BassSub)
	r.Register(&BassRolling)
	r.Register(&ArpFlowing)
	r.Register(&ArpEpic)
	r.Register(&ArpStaccato)
	r.Register(&MelodyExpressive)
	r.Register(&MelodyHypnotic)
	r.Register(&ChordPadGroove)
	r.Register(&ChordPadRolling)
	r.Register(&ChordPadSub)
	r.Register(&LeadStab)
	return r
}

// DefaultForType returns the default profile for a given pattern type.
// Handles all 7 pattern types in the extended bundle.
func (r *Registry) DefaultForTypeExtended(patternType string) (*StyleProfile, error) {
	defaults := map[string]string{
		"bassline":         "bass_progressive",
		"bassline_rolling": "bass_rolling",
		"bassline_sub":     "bass_sub",
		"arpeggio":         "arp_flowing",
		"melody":           "melody_expressive",
		"chord_pad":        "chord_pad_groove",
		"lead_stab":        "lead_stab",
	}
	name, ok := defaults[patternType]
	if !ok {
		return nil, fmt.Errorf("no default profile for pattern type %q", patternType)
	}
	p, ok := r.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not registered", name)
	}
	return p, nil
}

func (r *Registry) Register(p *StyleProfile) {
	r.profiles[p.Name] = p
}

func (r *Registry) LoadForType(patternType, profileName string) (*StyleProfile, error) {
	p, ok := r.profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("unknown style profile %q", profileName)
	}
	if p.PatternType != patternType {
		return nil, fmt.Errorf("profile %q is for %s, not %s", profileName, p.PatternType, patternType)
	}
	return p, nil
}

func (r *Registry) DefaultForType(patternType string) (*StyleProfile, error) {
	defaults := map[string]string{
		"bassline": "bass_progressive",
		"arpeggio": "arp_flowing",
		"melody":   "melody_expressive",
	}
	name, ok := defaults[patternType]
	if !ok {
		return nil, fmt.Errorf("no default profile for pattern type %q", patternType)
	}
	return r.LoadForType(patternType, name)
}
