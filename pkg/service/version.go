package service

import (
	"runtime/debug"
	"sync"
)

type VersionService struct {
	prettyVersion string
	once          sync.Once
}

func NewVersionService() *VersionService {
	return &VersionService{}
}

func (s *VersionService) GetRenderVersion() string {
	return "5.0.0"
}

func (s *VersionService) GetPrettyVersion() string {
	s.once.Do(func() {
		var (
			revision = "<no vcs info in build>"
			time     string
			modified = true
		)

		info, ok := debug.ReadBuildInfo()
		if !ok {
			s.prettyVersion = revision + "+dirty"
			return
		}

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.time":
				time = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			default:
				continue
			}
		}

		if modified {
			revision += "+dirty"
		}
		if time != "" {
			revision += " (committed " + time + ")"
		}
		if info.GoVersion != "" {
			revision += " (" + info.GoVersion + ")"
		}
		s.prettyVersion = revision
	})
	return s.prettyVersion
}
