package pixelysia

// version is the version of the running binary. Release builds inject the
// Git tag at link time via:
//
//	-ldflags "-X pixelysia/internal/pixelysia.version=vX.Y.Z"
//
// Locally built development binaries report "dev".
var version = "dev"

// Version returns the version of the running binary.
func Version() string {
	return version
}
