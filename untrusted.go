package vars

// ⛔ Some variables hold data this machine did not write: a module's
// result, a gathered fact, a value a target chose. A consumer that
// RE-RENDERS variables (go-ansible/playbook resolves them until they
// settle, emulating Ansible's lazy variables) must not re-render those,
// or a managed host's data is evaluated on the control node.
//
// The marking lives here, beside the variables, rather than being
// inferred from the LAYER a variable sits in. That was the first
// attempt and it was wrong in both directions: include_vars shares the
// Facts layer with set_fact, and real Ansible TEMPLATES an included
// vars file (it is a file the author named) while leaving a module's
// result alone. Treating the whole layer as untrusted stopped
// `include_vars` resolving `greeting: "Hello {{ who }}"` -- measured:
// real gives "Hello world", that version gave "Hello {{ who }}".
//
// Provenance is a property of the VALUE, so it is recorded per key.

// MarkUntrusted records that key holds data from outside this playbook.
func (c *Context) MarkUntrusted(key string) {
	if c.untrusted == nil {
		c.untrusted = map[string]bool{}
	}
	c.untrusted[key] = true
}

// Untrusted returns the keys marked by MarkUntrusted. The map is the
// Context's own: callers read it, they do not keep it.
func (c *Context) Untrusted() map[string]bool { return c.untrusted }

// IsUntrusted reports whether key was marked.
func (c *Context) IsUntrusted(key string) bool { return c.untrusted[key] }
