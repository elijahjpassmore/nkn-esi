/*
Copyright © 2021 Ecogy Energy

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

package cmd

import (
	"encoding/json"
	"github.com/elijahjpassmore/nkn-esi/api/esi"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

// registryInfo is the registry config details.
var registryInfo esi.DerRegistryInfo

// registryCmd represents the registry command.
var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage Registry instances.",
	Long:  `Manage Registry instances.`,
	Args:  cobra.ExactArgs(1),
}

// init initializes registry.go.
func init() {
	rootCmd.AddCommand(registryCmd)
}

// readRegistryConfig opens and reads the given registry config into a DerRegistryInfo struct.
func readRegistryConfig(registryPath string) error {
	registryFile, err := os.Open(registryPath)
	if err != nil {
		return err
	}
	defer registryFile.Close()

	byteValue, err := ioutil.ReadAll(registryFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &registryInfo)
	if err != nil {
		return err
	}

	return nil
}
