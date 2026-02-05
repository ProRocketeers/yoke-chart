package resources

// should be possible to override at build time with `go build -ldflags "-X resources.Version=1.0.0`, then it will be baked into the output
var (
	// here we can set the default, but it will be best to keep it in sync with the released version, so git tag represents the binary as well
	Version = "1.3.0"
)
