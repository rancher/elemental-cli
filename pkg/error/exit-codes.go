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

// Exit code 10: Error closing a file
const CloseFile = 10

// Exit code 11: Error running a command
const CommandRun = 11

// Exit code 12: Error copying data
const CopyData = 12

// Exit code 13: Error copying a file
const CopyFile = 13

// Exit code 14: Wrong cosign flags used in cmd
const CosignWrongFlags = 14

// Exit code 15: Error creating a dir
const CreateDir = 15

// Exit code 16: Error creating a file
const CreateFile = 16

// Exit code 17: Error creating a temporal dir
const CreateTempDir = 17

// Exit code 18: Error dumping the source
const DumpSource = 18

// Exit code 19: Error creating a gzip writer
const GzipWriter = 19

// Exit code 20: Error trying to identify the source
const IdentifySource = 20

// Exit code 21: Error calling mkfs
const MKFSCall = 21

// Exit code 22: There is not packages for the given architecture
const NoPackagesForArch = 22

// Exit code 23: No luet repositories configured
const NoReposConfigured = 23

// Exit code 24: Error opening a file
const OpenFile = 24

// Exit code 25: Output file already exists
const OutFileExists = 25

// Exit code 26: Error reading the build config
const ReadingBuildConfig = 26

// Exit code 27:  Error reading the build-disk config
const ReadingBuildDiskConfig = 27

// Exit code 28: Error running stat on a file
const StatFile = 28

// Exit code 29: Error creating a tar archive
const TarHeader = 29

// Exit code 30: Error truncating a file
const TruncateFile = 30

// Exit code 255: Unknown error
const Unknown = 255
