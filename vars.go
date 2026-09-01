// Package vars implements Ansible's variable precedence: a fixed ladder
// of named layers (role defaults, inventory, facts, play/role/block/task
// vars, registered vars, role params, extra vars) merged low-to-high so
// that a value set in a higher layer always wins.
//
// This does not itself know about roles, plays, or tasks — playbook
// assigns each layer's content as it walks the play; this package only
// owns the merge order.
package vars

import "sort"

// Layer names one rung of Ansible's variable precedence ladder, lowest
// first. This is a simplified but order-faithful subset of the ~22
// levels documented for ansible-core: it keeps every distinction that
// changes real-world playbook behavior and collapses the rest (e.g.
// vars_prompt/vars_files fold into PlayVars — they all resolve to the
// same play-scoped map before a task runs).
type Layer int

const (
	RoleDefaults Layer = iota // role defaults/main.yml — the floor
	Inventory                 // merged inventory group_vars + host_vars
	Facts                     // gathered facts (ansible_facts.*) + flattened ansible_* aliases
	PlayVars                  // play vars:, vars_files:, vars_prompt:
	RoleVars                  // role vars/main.yml
	BlockVars                 // block-level vars:
	TaskVars                  // task-level vars:, loop item vars
	Registered                // register:, set_fact
	RoleParams                // role/include_role parameters
	ExtraVars                 // -e / --extra-vars — always wins
	numLayers
)

var layerNames = map[Layer]string{
	RoleDefaults: "role_defaults",
	Inventory:    "inventory",
	Facts:        "facts",
	PlayVars:     "play_vars",
	RoleVars:     "role_vars",
	BlockVars:    "block_vars",
	TaskVars:     "task_vars",
	Registered:   "registered",
	RoleParams:   "role_params",
	ExtraVars:    "extra_vars",
}

func (l Layer) String() string {
	if n, ok := layerNames[l]; ok {
		return n
	}
	return "unknown"
}

// Context holds one host/task's variable state across every precedence
// layer. The zero value is not usable — use New.
type Context struct {
	layers [numLayers]map[string]any
}

// New returns an empty Context.
func New() *Context {
	c := &Context{}
	for i := range c.layers {
		c.layers[i] = map[string]any{}
	}
	return c
}

// Set replaces layer's whole variable map.
func (c *Context) Set(layer Layer, vals map[string]any) {
	m := map[string]any{}
	for k, v := range vals {
		m[k] = v
	}
	c.layers[layer] = m
}

// SetVar sets a single variable within layer.
func (c *Context) SetVar(layer Layer, key string, value any) {
	if c.layers[layer] == nil {
		c.layers[layer] = map[string]any{}
	}
	c.layers[layer][key] = value
}

// Merged flattens every layer into one map, low precedence first so
// higher layers overwrite lower ones on key conflict.
func (c *Context) Merged() map[string]any {
	out := map[string]any{}
	for l := Layer(0); l < numLayers; l++ {
		for k, v := range c.layers[l] {
			out[k] = v
		}
	}
	return out
}

// Get looks up key in the fully merged view.
func (c *Context) Get(key string) (any, bool) {
	v, ok := c.Merged()[key]
	return v, ok
}

// Which reports the highest-precedence layer that currently sets key,
// for diagnostics ("why did this variable win?").
func (c *Context) Which(key string) (Layer, bool) {
	for l := numLayers - 1; l >= 0; l-- {
		if _, ok := c.layers[l][key]; ok {
			return l, true
		}
	}
	return 0, false
}

// Child returns a copy of c suitable for a nested scope (a block inside
// a play, a task inside a block, one iteration of a loop): mutating the
// child's TaskVars/BlockVars/Registered layers never affects the parent.
func (c *Context) Child() *Context {
	child := New()
	for l := Layer(0); l < numLayers; l++ {
		for k, v := range c.layers[l] {
			child.layers[l][k] = v
		}
	}
	return child
}

// Layers returns the layer set in ascending precedence order, for
// callers that want to render the whole ladder (e.g. `ansible -e` style
// debugging tools).
func Layers() []Layer {
	out := make([]Layer, 0, numLayers)
	for l := Layer(0); l < numLayers; l++ {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// InjectFacts returns facts both nested under "ansible_facts" (the
// canonical form) and flattened as top-level "ansible_<name>" aliases
// (e.g. ansible_facts["os_family"] also becomes "ansible_os_family"),
// matching Ansible's fact-injection behavior so both spellings resolve
// in templates.
func InjectFacts(facts map[string]any) map[string]any {
	out := map[string]any{"ansible_facts": facts}
	for k, v := range facts {
		out["ansible_"+k] = v
	}
	return out
}
