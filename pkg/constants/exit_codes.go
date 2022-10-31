package constants

const (
	ExitUnknownError = iota + 10
	ExitCopyError
	ExitTempdirError
	ExitNoPackagesArchBuildDisk
	ExitNoRepoConfiguredBuildDisk
	ExitCreateDirError
	ExitCreateFileError
	ExitTruncateFileError
	ExitFailedFileCloseError
	ExitOpenFileError
	ExitStatFileError
	ExitTarError
	ExitGzipError
	ExitParseSourceError
	ExitCosignError
	ExitConfigError
	ExitFileExists
	ExitFailedCratingPart
	ExitFailedRunningCmd
)
