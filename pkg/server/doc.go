// Package server exposes the drop-in GoMyAdmin HTTP server.
//
// Existing Go applications can mount AdminServer.Handler on any router that
// accepts http.Handler. Use Config.DatabaseURL or Config.Pool for the built-in
// PostgreSQL adapter, or Config.Store for custom databases and ORMs.
package server
