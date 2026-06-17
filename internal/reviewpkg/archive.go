package reviewpkg

import "github.com/TrebuchetDynamics/research-forge/internal/project"

func Archive(packagePath, archivePath string) error { return project.Archive(packagePath, archivePath) }

func Restore(archivePath, destination string) error { return project.Restore(archivePath, destination) }
