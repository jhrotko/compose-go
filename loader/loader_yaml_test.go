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

package loader

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"gotest.tools/v3/assert"
)

func TestParseYAMLFiles(t *testing.T) {
	model, err := loadYamlModel(context.TODO(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Filename: "test.yaml",
				Content: []byte(`
services:
  test:
    image: foo
    command: echo hello
    init: true
`),
			},
			{Filename: "override.yaml",
				Content: []byte(`
services:
  test:
    image: bar
    command: echo world
    init: false
`)}}}, &Options{}, &cycleTracker{}, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, model, map[string]interface{}{
		"services": map[string]interface{}{
			"test": map[string]interface{}{
				"image":   "bar",
				"command": "echo world",
				"init":    false,
			},
		},
	})
}

func TestParseYAMLFilesInterpolateAfterMerge(t *testing.T) {
	model, err := loadYamlModel(
		context.TODO(), types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Filename: "test.yaml",
					Content: []byte(`
services:
  test:
    image: foo
    environment:
      my_env: ${my_env?my_env must be set}
`),
				},
				{
					Filename: "override.yaml",
					Content: []byte(`
services:
  test:
    image: bar
    environment:
      my_env: ${my_env:-default}
`),
				},
			},
		}, &Options{}, &cycleTracker{}, nil,
	)
	assert.NilError(t, err)
	assert.DeepEqual(
		t, model, map[string]interface{}{
			"services": map[string]interface{}{
				"test": map[string]interface{}{
					"image":       "bar",
					"environment": []any{"my_env=${my_env:-default}"},
				},
			},
		},
	)
}

func TestParseYAMLFilesMergeOverride(t *testing.T) {
	model, err := loadYamlModel(context.TODO(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Filename: "override.yaml",
				Content: []byte(`
services:
  base:
    configs:
      - source: credentials
        target: /credentials/file1
  x: &x
    extends:
      base
    configs: !override
      - source: credentials
        target: /literally-anywhere-else

  y:
    <<: *x

configs:
  credentials:
    content: |
      dummy value
`)},
		}}, &Options{}, &cycleTracker{}, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, model, map[string]interface{}{
		"configs": map[string]interface{}{"credentials": map[string]interface{}{"content": string("dummy value\n")}},
		"services": map[string]interface{}{
			"base": map[string]interface{}{
				"configs": []interface{}{
					map[string]interface{}{"source": string("credentials"), "target": string("/credentials/file1")},
				},
			},
			"x": map[string]interface{}{
				"configs": []interface{}{
					map[string]interface{}{"source": string("credentials"), "target": string("/literally-anywhere-else")},
				},
			},
			"y": map[string]interface{}{
				"configs": []interface{}{
					map[string]interface{}{"source": string("credentials"), "target": string("/literally-anywhere-else")},
				},
			},
		},
	})
}

func TestParseYAMLFilesMergeOverrideArray(t *testing.T) {
	model, err := loadYamlModel(context.TODO(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Filename: "override.yaml",
				Content: []byte(`
x-app:
  volumes: &app-volumes
    - /data/app:/app/data
services:
  app:
    image: myapp:latest
    volumes:
      - *app-volumes
      - /logs/app:/app/logs
`)},
		}}, &Options{}, &cycleTracker{}, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, model, map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"image": "myapp:latest",
				"volumes": []interface{}{
					map[string]interface{}{
						"bind": map[string]interface{}{
							"create_host_path": bool(true),
						},
						"source": string("/data/app"),
						"target": string("/app/data"), // should merge /data/app:/app/data
						"type":   string("bind"),
					},
					map[string]interface{}{
						"bind": map[string]interface{}{
							"create_host_path": bool(true),
						},
						"source": string("/logs/app"),
						"target": string("/app/logs"),
						"type":   string("bind"),
					},
				},
			},
		},
		"x-app": map[string]interface{}{
			"volumes": []interface{}{
				string("/data/app:/app/data"),
			},
		},
	})
}

func TestParseYAMLFilesMergeMapArray(t *testing.T) {
	model, err := loadYamlModel(context.TODO(), types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Filename: "override.yaml",
				Content: []byte(`
x-app: &app-volumes
  image: alpine
  volumes: 
    - /data/app:/app/data
services:
  app:
    image: python
    <<: *app-volumes
    volumes:
      - /logs/app:/app/logs
`)},
		}}, &Options{}, &cycleTracker{}, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, model, map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"image": "alpine", // should override
				"volumes": []interface{}{
					map[string]interface{}{
						"bind": map[string]interface{}{
							"create_host_path": bool(true),
						},
						"source": string("/logs/app"),
						"target": string("/app/logs"),
						"type":   string("bind"),
					},
					map[string]interface{}{
						"bind": map[string]interface{}{
							"create_host_path": bool(true),
						},
						"source": string("/data/app"),
						"target": string("/app/data"), // should merge /data/app:/app/data
						"type":   string("bind"),
					},
				},
			},
		},
		"x-app": map[string]interface{}{
			"image": "alpine",
			"volumes": []interface{}{
				string("/data/app:/app/data"),
			},
		},
	})
}
