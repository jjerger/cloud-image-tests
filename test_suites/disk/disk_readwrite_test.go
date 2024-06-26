// Copyright 2024 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package disk

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-image-tests/utils"
)

func TestDiskReadWrite(t *testing.T) {
	if runtime.GOOS == "linux" {
		testDiskReadWriteLinux(t)
	} else {
		testDiskReadWriteWindows(t)
	}
}

func testDiskReadWriteLinux(t *testing.T) {
	testFile := "/var/test.txt"
	newTestFile := "/var/testnew.txt"
	content := "Test File Content"
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file at path %s: error %v", testFile, err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to file: err %v", err)
	}
	if err = os.Rename(testFile, newTestFile); err != nil {
		t.Fatalf("failed to move file: err %v", err)
	}

	renamedFileBytes, err := os.ReadFile(newTestFile)
	if err != nil {
		t.Fatalf("failed to read contents of new file: error %v", err)
	}

	renamedFileContents := string(renamedFileBytes)
	if renamedFileContents != content {
		t.Fatalf("Moved file does not contain expected content. Expected: '%s', Actual: '%s'", content, renamedFileContents)
	}
}

func testDiskReadWriteWindows(t *testing.T) {
	testFile := "C:\\test.txt"
	newTestFile := "C:\\testnew.txt"
	content := "Test File Content"
	command := fmt.Sprintf("Set-Content %s \"%s\"", testFile, content)
	utils.FailOnPowershellFail(command, "Error writing file", t)

	command = fmt.Sprintf("Move-Item -Force %s %s", testFile, newTestFile)
	utils.FailOnPowershellFail(command, "Error moving file", t)

	command = fmt.Sprintf("Get-Content %s", newTestFile)
	output, err := utils.RunPowershellCmd(command)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	if !strings.Contains(output.Stdout, content) {
		t.Fatalf("Moved file does not contain expected content. Expected: '%s', Actual: '%s'", content, output.Stdout)
	}
}
