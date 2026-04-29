package styleprofile

import "fmt"

type Registry struct {
	profiles map[string]*StyleProfile
}

func NewRegistry() *Registry {
	r := &Registry{profiles: make(map[string]*StyleProfile)}
	r.Register(&BassProgressive)
	r.Register(&ArpFlowing)
	r.Register(&MelodyExpressive)
	return r
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
