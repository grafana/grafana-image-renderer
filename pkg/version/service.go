package version

import "runtime/debug"

// ServiceVersion returns the service's version data. It is complete only if `-buildvcs` is passed to the Go build command.
func ServiceVersion() string {
	var (
		revision = "<no vcs info in build>"
		time     string
		modified = true
	)

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return revision + "+dirty"
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
	return revision
}
