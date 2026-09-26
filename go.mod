module github.com/aleogr/identity

go 1.27.0

toolchain go1.27.1

require github.com/aleogr/identity/core v0.0.0

// The root module is the server, never imported as a library, so it may
// replace its own libraries with their directories. Builds outside the
// workspace (ko, Dependabot) resolve them through these lines.
replace github.com/aleogr/identity/core => ./core
