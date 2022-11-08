/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package error

const (
	// Wrong cosign flags used in cmd
	CosignWrongFlags = iota + 10
	// Error reading the build config
	ReadingBuildConfig
	// Error reading the build-disk config
	ReadingBuildDiskConfig
	// Output file already exists
	OutFileExists
	// There is not packages for the given architecture
	NoPackagesForArch
	// No luet repositories configured
	NoReposConfigured
	// Error creating a temporal dir
	CreateTempDir
	// Error creating a dir
	CreateDir
	// Error trying to identify the source
	IdentifySource
	// Error dumping the source
	DumpSource
	CreatePart
	// Error running a command
	CommandRun
	// Error copying a file
	CopyFile
	CreateFinalImage
	// Error opening a file
	OpenFile
	// Error running stat on a file
	StatFile
	// Error creating a file
	CreateFile
	// Error truncating a file
	TruncateFile
	// Error closing a file
	CloseFile
	// Error creating a gzip writer
	GzipWriter
	// Error creating a tar archive
	TarHeader
	// Error copying data
	CopyData
	// Error calling mkfs
	MKFSCall
	// Unknown error
	Unknown
)
