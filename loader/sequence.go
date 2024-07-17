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
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/tree"
	"gopkg.in/yaml.v3"
)

type SequenceProcessor struct {
	target interface{}
	paths  []tree.Path
}

// UnmarshalYAML implement yaml.Unmarshaler
func (p *SequenceProcessor) UnmarshalYAML(value *yaml.Node) error {
	resolved, err := p.resolveSequence(value, tree.NewPath())
	if err != nil {
		return err
	}
	return resolved.Decode(p.target)
}

// resolveSequence detects `!reset` tag being set on yaml nodes and record position in the yaml tree
func (p *SequenceProcessor) resolveSequence(node *yaml.Node, path tree.Path) (*yaml.Node, error) {
	if strings.Contains(path.String(), ".<<") {
		// If the path contains "<<", removing the "<<" element and merging the path
		path = tree.NewPath(strings.Replace(path.String(), ".<<", "", 1))
		// // if we enconter the merge, first we resolve the node
		// resolved, err := p.resolveSequence(node.Alias, path)
		// if err != nil {
		// 	return nil, err
		// }
		// // we can only solve the merge at the end

	}
	// If the node is an alias, We need to process the alias field in order to consider the !override and !reset tags
	if node.Kind == yaml.AliasNode {
		fmt.Printf("\nnode content: %v\n", node.Content)
		resolved, err := p.resolveSequence(node.Alias, path)
		if err != nil {
			return nil, err
		}
		return resolved, nil
	}

	switch node.Kind {
	case yaml.SequenceNode:
		var nodes []*yaml.Node
		for idx, v := range node.Content {
			// if current node, v, is an sequence and has an alias which is a sequence
			// we need to flatten the array

			next := path.Next(strconv.Itoa(idx))
			resolved, err := p.resolveSequence(v, next)
			if err != nil {
				return nil, err
			}
			if resolved != nil {
				nodes = append(nodes, resolved)
			}
		}
		node.Content = flatten(nodes)
	case yaml.MappingNode:
		var key string
		var nodes []*yaml.Node
		var merge []*yaml.Node
		for idx, v := range node.Content {
			if idx%2 == 0 {
				key = v.Value
			} else {
				resolved, err := p.resolveSequence(v, path.Next(key))
				if err != nil {
					return nil, err
				}
				if resolved != nil {
					if key == "<<" {
						merge = append(merge, resolved)
					} else {
						nodes = append(nodes, node.Content[idx-1], resolved)
					}
				}
			}
		}
		mergeNodes(merge, nodes)
		// update nodes
		node.Content = nodes
	}
	return node, nil
}

// Apply finds the go attributes matching recorded paths and reset them to zero value
func (p *SequenceProcessor) Apply(target any) error {
	return p.applyNullOverrides(target, tree.NewPath())
}

func (p *SequenceProcessor) applyNullOverrides(target any, path tree.Path) error {
	switch v := target.(type) {
	case map[string]any:
	KEYS:
		for k, e := range v {
			next := path.Next(k)
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					delete(v, k)
					continue KEYS
				}
			}
			err := p.applyNullOverrides(e, next)
			if err != nil {
				return err
			}
		}
	case []any:
	ITER:
		for i, e := range v {
			next := path.Next(fmt.Sprintf("[%d]", i))
			for _, pattern := range p.paths {
				if next.Matches(pattern) {
					continue ITER
					// TODO(ndeloof) support removal from sequence
				}
			}
			err := p.applyNullOverrides(e, next)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func mergeNodes(merge []*yaml.Node, nodes []*yaml.Node) {
	// merge after evaluating the level of the tree
	for _, anchor := range merge {
		// app-volumes
		var key string
		for idx, anchorNode := range anchor.Content {

			if idx%2 == 0 {
				key = anchorNode.Value // volumes
			} else {
				switch anchorNode.Kind {
				case yaml.SequenceNode:
					for i, n := range nodes {
						if i%2 == 0 && n.Value == key {
							// content will be the next node
							// need to deal with overflow (?)
							c := nodes[i+1]
							if c.Kind != yaml.SequenceNode {
								// c should also be a sequence? if not we ignore
								break
							}
							// merging sequences v and c ...
							for _, v := range anchorNode.Content {
								found := false
								for _, el := range c.Content {
									// might be comparing maps T.T, Value wont work
									if v.Value == el.Value {
										found = true
									}
								}
								if !found {
									c.Content = append(c.Content, v)
								}
							}
							// if the node already exists in anchorNode content, do nothing
						}
					}
				case yaml.MappingNode:
					break
				default:
					for i, n := range nodes {
						if i%2 == 0 && n.Value == key {
							c := nodes[i+1]
							if c.Kind == anchorNode.Kind {
								c.Value = anchorNode.Value
							}
							// not the same type, skipping
							break
						}
					}
				}
			}
		}
	}
}

func flatten(nodes []*yaml.Node) []*yaml.Node {
	flattened := make([]*yaml.Node, 0)

	for _, node := range nodes {
		if node.Kind == yaml.SequenceNode {
			flattenedSubArray := flatten(node.Content)
			flattened = append(flattened, flattenedSubArray...)
		} else {
			flattened = append(flattened, node)
		}
	}

	return flattened
}
