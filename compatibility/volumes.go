/*
   Copyright 2020 The Compose Specification Authors.

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

package compatibility

import "github.com/compose-spec/compose-go/types"

func (c *WhiteList) CheckVolumeConfigDriver(config *types.VolumeConfig) {
	if !c.supported("volumes.driver") && config.Driver != "" {
		config.Driver = ""
		c.error("volumes.driver")
	}
}

func (c *WhiteList) CheckVolumeConfigDriverOpts(config *types.VolumeConfig) {
	if !c.supported("volumes.driver_opts") && len(config.DriverOpts) != 0 {
		config.DriverOpts = nil
		c.error("volumes.driver_opts")
	}
}

func (c *WhiteList) CheckVolumeConfigExternal(config *types.VolumeConfig) {
	if !c.supported("volumes.external") && config.External.External {
		config.External.External = false
		c.error("volumes.external")
	}
}

func (c *WhiteList) CheckVolumeConfigLabels(config *types.VolumeConfig) {
	if !c.supported("volumes.labels") && len(config.Labels) != 0 {
		config.Labels = nil
		c.error("volumes.labels")
	}
}