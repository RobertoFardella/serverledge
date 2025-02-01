package container

// RuntimeInfo contains information about a supported function runtime env.
type RuntimeInfo struct {
	Image         string
	InvocationCmd []string
}

const CUSTOM_RUNTIME = "custom"

var refreshedImages = map[string]bool{}

var RuntimeToInfo = map[string]RuntimeInfo{
	"python310":              {"roberto1999/serverledge-python310", []string{"python", "/entrypoint.py"}},
	"nodejs17":               {"grussorusso/serverledge-nodejs17", []string{"node", "/entrypoint.js"}},
	"nodejs17-multithread":   {"roberto1999/serverledge_multithread-nodejs17", []string{"node", "/entrypoint.js"}},
	"nodejs18":               {"roberto1999/serverledge-nodejs18", []string{"node", "/entrypoint.js"}},
	"nodejs18-single_thread": {"roberto1999/serverledge_base-nodejs18", []string{"node", "/entrypoint.js"}},
	"nodejs17ng":             {"grussorusso/serverledge-nodejs17ng", []string{}},
}
