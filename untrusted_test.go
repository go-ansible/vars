package vars

import "testing"

// A mark follows the variable into a child scope: a task scope must not
// re-render a module result either.
func TestChildInheritsUntrustedMarks(t *testing.T) {
	c := New()
	c.SetVar(Registered, "r", map[string]any{"stdout": "{{ 7*7 }}"})
	c.MarkUntrusted("r")

	child := c.Child()
	if !child.IsUntrusted("r") {
		t.Error("a child scope lost the mark; it would re-render a module result")
	}
	// And a mark made on the CHILD does not leak back, matching how the
	// child's own layers do not.
	child.MarkUntrusted("only_here")
	if c.IsUntrusted("only_here") {
		t.Error("a child's mark leaked into its parent")
	}
}

// An unmarked variable stays unmarked -- the point of recording
// provenance per key is that a layer may hold both kinds. include_vars
// and set_fact share one.
func TestMarksArePerKeyNotPerLayer(t *testing.T) {
	c := New()
	c.SetVar(Facts, "from_include", "Hello {{ who }}")
	c.SetVar(Facts, "from_module", "{{ 7*7 }}")
	c.MarkUntrusted("from_module")

	if c.IsUntrusted("from_include") {
		t.Error("an included value was marked; real templates an author's vars file")
	}
	if !c.IsUntrusted("from_module") {
		t.Error("a module result was not marked")
	}
	if got := len(c.Untrusted()); got != 1 {
		t.Errorf("Untrusted() holds %d keys, want 1", got)
	}
}
