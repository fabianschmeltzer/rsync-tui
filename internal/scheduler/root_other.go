//go:build !unix

package scheduler

func isRoot() bool {
	return false
}
