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

// provides a custom error interface and exit codes to use on the elemental-cli
package error

//
// Provided exit codes for elemental-cli

// Exit code 10: Wrong cosign flags used in cmd
const CosignWrongFlags = 10

// Exit code 11: Error reading the build config
const ReadingBuildConfig = 11

// Exit code 12:  Error reading the build-disk config
const ReadingBuildDiskConfig = 12

// Exit code 13: Output file already exists
const OutFileExists = 13

// Exit code 14: There is not packages for the given architecture
const NoPackagesForArch = 14

// Exit code 15: No luet repositories configured
const NoReposConfigured = 15

// Exit code 16: Error creating a temporal dir
const CreateTempDir = 16

// Exit code 17: Error creating a dir
const CreateDir = 17

// Exit code 18: Error trying to identify the source
const IdentifySource = 18

// Exit code 19: Error dumping the source
const DumpSource = 19

// Exit code 20: Error running a command
const CommandRun = 20

// Exit code 21: Error copying a file
const CopyFile = 21

// Exit code 22: Error opening a file
const OpenFile = 22

// Exit code 23: Error running stat on a file
const StatFile = 23

// Exit code 24: Error creating a file
const CreateFile = 24

// Exit code 25: Error truncating a file
const TruncateFile = 25

// Exit code 26: Error closing a file
const CloseFile = 26

// Exit code 27: Error creating a gzip writer
const GzipWriter = 27

// Exit code 28: Error creating a tar archive
const TarHeader = 28

// Exit code 29: Error copying data
const CopyData = 29

// Exit code 30: Error calling mkfs
const MKFSCall = 30

// Exit code 255: Unknown error
const Unknown = 255
