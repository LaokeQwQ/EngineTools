//go:build bindings

package main

// Binding generation runs a temporary executable without opening the application.
func shouldElevate() bool { return false }
