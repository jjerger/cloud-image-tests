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

package packagevalidation

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-image-tests/utils"
)

// osPackage defines the rules for expected installed packages.
type osPackage struct {
	// name is the name of the package, a package could have alternative names
	// depending on the distro, see alternatives field.
	name string

	// shouldNotBeInstalled defines if we are checking if the package should or
	// should not be installed.
	shouldNotBeInstalled bool

	// alternatives are alternative names, a package can be named differently
	// depending on the distribution.
	alternatives []string

	// imagesSkip are the image name matching expression for images we don't want
	// to check this package rule.
	// The expression matching is applied with strings.Contains() so if the image
	// name contains the substring it will match.
	imagesSkip []string

	// images is the oposite of imagesSkip and defines the image name matching
	// expression of the images this rule must apply.
	// The expression matching is applied with strings.Contains() so if the image
	// name contains the substring it will match.
	images []string
}

func TestStandardPrograms(t *testing.T) {
	image, err := utils.GetMetadata(utils.Context(t), "instance", "image")
	if err != nil {
		t.Fatalf("couldn't get image from metadata")
	}
	if strings.Contains(image, "sles") || strings.Contains(image, "suse") {
		// SLES/SUSE does not have the Google Cloud SDK installed.
		t.Skip("Cloud SDK Not supported on SLES/SUSE")
	}
	if strings.Contains(image, "cos") {
		// COS does not have the Google Cloud SDK installed.
		t.Skip("Cloud SDK Not supported on COS")
	}

	cmd := exec.Command("gcloud", "-h")
	cmd.Start()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("gcloud not installed properly")
	}
	cmd = exec.Command("gsutil", "help")
	cmd.Start()
	err = cmd.Wait()
	if err != nil {
		t.Fatalf("gsutil not installed properly")
	}
}

func TestGuestPackages(t *testing.T) {
	utils.LinuxOnly(t)
	image, err := utils.GetMetadata(utils.Context(t), "instance", "image")

	if err != nil {
		t.Fatalf("couldn't determine image from metadata")
	}

	// What command to list all packages
	listPkgs := func() ([]string, error) {
		return nil, fmt.Errorf("could not determine how to list installed packages")
	}
	switch {
	case utils.CheckLinuxCmdExists("rpm"):
		listPkgs = func() ([]string, error) {
			o, err := exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\n").Output()
			return strings.Split(string(o), "\n"), err
		}
	case utils.CheckLinuxCmdExists("dpkg-query") && utils.CheckLinuxCmdExists("snap"):
		listPkgs = func() ([]string, error) {
			var pkgs []string
			dpkgout, err := exec.Command("dpkg-query", "-W", "--showformat", "${Package}\n").Output()
			if err != nil {
				return nil, err
			}
			pkgs = append(pkgs, strings.Split(string(dpkgout), "\n")...)
			// Snap format name regexp source:
			// https://snapcraft.io/docs/the-snap-format
			snapname := regexp.MustCompile("[a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9]")
			snapout, err := exec.Command("snap", "list").Output()
			if err != nil {
				return nil, err
			}
			for i, line := range strings.Split(string(snapout), "\n") {
				if i == 0 {
					continue // Skip header
				}
				if pkg := snapname.FindString(line); pkg != "" {
					pkgs = append(pkgs, pkg)
				}
			}
			return pkgs, nil
		}
	case utils.CheckLinuxCmdExists("dpkg-query"):
		listPkgs = func() ([]string, error) {
			o, err := exec.Command("dpkg-query", "-W", "--showformat", "${Package}\n").Output()
			return strings.Split(string(o), "\n"), err
		}
	}

	if strings.Contains(image, "cos") {
		listPkgs = func() ([]string, error) {
			o, err := os.ReadFile("/etc/cos-package-info.json")
			pkgs := []string{}
			for _, line := range strings.Split(string(o), "\n") {
				if strings.Contains(line, "name\": ") {
					pkgField := strings.Split(line, ":")[1]
					pkg := strings.Split(pkgField, "\"")[1]
					pkgs = append(pkgs, pkg)
				}
			}
			return pkgs, err
		}
	}

	pkgs := []*osPackage{
		{
			name: "google-guest-agent",
		},
		{
			name: "google-osconfig-agent",
		},
		{
			name:       "google-compute-engine",
			imagesSkip: []string{"sles", "suse", "cos"},
		},
		{
			name:   "google-guest-configs",
			images: []string{"sles", "suse", "cos"},
		},
		{
			name:   "google-guest-oslogin",
			images: []string{"sles", "suse"},
		},
		{
			name:   "oslogin",
			images: []string{"cos"},
		},
		{
			name:       "gce-disk-expand",
			imagesSkip: []string{"sles", "suse", "ubuntu", "cos"},
		},
		{
			name:   "cloud-disk-resize",
			images: []string{"cos"},
		},
		{
			name:       "google-cloud-cli",
			imagesSkip: []string{"sles", "suse", "ubuntu-1604", "ubuntu-pro-1604", "cos"},
		},
		{
			name:       "google-compute-engine-oslogin",
			imagesSkip: []string{"sles", "suse", "cos"},
		},
		{
			name:   "epel-release",
			images: []string{"centos-7", "rhel-7"},
		},
		{
			name:   "haveged",
			images: []string{"debian"},
		},
		{
			name:   "net-tools",
			images: []string{"debian", "cos"},
		},
		{
			name:   "google-cloud-packages-archive-keyring",
			images: []string{"debian"},
		},
		{
			name:   "isc-dhcp-client",
			images: []string{"debian"},
		},
		{
			name:                 "cloud-initramfs-growroot",
			shouldNotBeInstalled: true,
			images:               []string{"debian"},
		},
	}

	installedList, err := listPkgs()
	if err != nil {
		t.Errorf("Failed to execute list packages command: %v", err)
		return
	}

	installedMap := make(map[string]bool)
	for _, curr := range installedList {
		installedMap[curr] = true
	}

	for _, curr := range pkgs {
		skipPackage := false
		for _, skipExpression := range curr.imagesSkip {
			if strings.Contains(image, skipExpression) {
				skipPackage = true
				break
			}
		}

		imageMatched := len(curr.images) == 0
		for _, matchExpression := range curr.images {
			if strings.Contains(image, matchExpression) {
				imageMatched = true
				break
			}
		}

		if skipPackage || !imageMatched {
			continue
		}

		packageInstalled := false
		packageNames := []string{curr.name}
		packageNames = append(packageNames, curr.alternatives...)

		for _, currPackage := range packageNames {
			if _, found := installedMap[currPackage]; found {
				packageInstalled = true
				break
			}
		}

		if !curr.shouldNotBeInstalled != packageInstalled {
			t.Errorf("package %s has wrong installation state, got (shouldNotBeInstalled: %t, packageInstalled: %t)",
				curr.name, curr.shouldNotBeInstalled, packageInstalled)
		}
	}
}
