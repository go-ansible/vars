package vars

import "testing"

func TestPrecedenceOrder(t *testing.T) {
	c := New()
	c.SetVar(RoleDefaults, "x", "default")
	c.SetVar(Inventory, "x", "inventory")
	c.SetVar(Facts, "x", "facts")
	c.SetVar(PlayVars, "x", "play")
	c.SetVar(RoleVars, "x", "role")
	c.SetVar(BlockVars, "x", "block")
	c.SetVar(TaskVars, "x", "task")
	c.SetVar(Registered, "x", "registered")
	c.SetVar(RoleParams, "x", "roleparams")
	c.SetVar(ExtraVars, "x", "extra")

	if got, _ := c.Get("x"); got != "extra" {
		t.Fatalf("Get(x) = %v, want extra (extra_vars must always win)", got)
	}

	layer, ok := c.Which("x")
	if !ok || layer != ExtraVars {
		t.Fatalf("Which(x) = %v,%v want ExtraVars,true", layer, ok)
	}

	// Remove the extra_vars layer and confirm the next-highest wins.
	c.Set(ExtraVars, nil)
	if got, _ := c.Get("x"); got != "roleparams" {
		t.Fatalf("Get(x) after clearing extra_vars = %v, want roleparams", got)
	}
}

func TestUnsetVariableFallsThrough(t *testing.T) {
	c := New()
	c.SetVar(RoleDefaults, "only_default", "d")
	c.SetVar(ExtraVars, "only_extra", "e")
	if v, _ := c.Get("only_default"); v != "d" {
		t.Fatalf("only_default = %v", v)
	}
	if v, _ := c.Get("only_extra"); v != "e" {
		t.Fatalf("only_extra = %v", v)
	}
	if _, ok := c.Get("nowhere"); ok {
		t.Fatal("Get(nowhere) should report not-found")
	}
}

func TestChildIsolation(t *testing.T) {
	parent := New()
	parent.SetVar(PlayVars, "shared", "parent")

	child := parent.Child()
	child.SetVar(TaskVars, "shared", "child-override")
	child.SetVar(TaskVars, "only_child", "x")

	if got, _ := child.Get("shared"); got != "child-override" {
		t.Fatalf("child shared = %v, want child-override", got)
	}
	if got, _ := parent.Get("shared"); got != "parent" {
		t.Fatalf("parent shared = %v, want parent (child mutation must not leak)", got)
	}
	if _, ok := parent.Get("only_child"); ok {
		t.Fatal("only_child leaked into parent")
	}
}

func TestInjectFacts(t *testing.T) {
	facts := map[string]any{"os_family": "Debian", "hostname": "web1"}
	injected := InjectFacts(facts)

	nested, ok := injected["ansible_facts"].(map[string]any)
	if !ok {
		t.Fatalf("ansible_facts missing or wrong type: %v", injected["ansible_facts"])
	}
	if nested["os_family"] != "Debian" {
		t.Fatalf("ansible_facts.os_family = %v", nested["os_family"])
	}
	if injected["ansible_os_family"] != "Debian" {
		t.Fatalf("ansible_os_family = %v, want Debian (flattened alias)", injected["ansible_os_family"])
	}
	if injected["ansible_hostname"] != "web1" {
		t.Fatalf("ansible_hostname = %v, want web1", injected["ansible_hostname"])
	}
}

func TestMergedIsASnapshot(t *testing.T) {
	c := New()
	c.SetVar(PlayVars, "x", 1)
	m := c.Merged()
	m["x"] = 2
	if got, _ := c.Get("x"); got != 1 {
		t.Fatalf("mutating Merged() result affected the Context: got %v", got)
	}
}

func TestLayerReturnsACopy(t *testing.T) {
	c := New()
	c.SetVar(RoleDefaults, "x", 1)

	snapshot := c.Layer(RoleDefaults)
	if snapshot["x"] != 1 {
		t.Fatalf("Layer(RoleDefaults)[\"x\"] = %v, want 1", snapshot["x"])
	}

	snapshot["x"] = 2
	if got, _ := c.Get("x"); got != 1 {
		t.Fatalf("mutating Layer()'s result affected the Context: got %v", got)
	}
}

func TestLayerEmptyByDefault(t *testing.T) {
	c := New()
	if got := c.Layer(RoleVars); len(got) != 0 {
		t.Fatalf("Layer(RoleVars) = %v, want empty on a fresh Context", got)
	}
}

func TestLayerString(t *testing.T) {
	cases := []struct {
		l    Layer
		want string
	}{
		{RoleDefaults, "role_defaults"},
		{Inventory, "inventory"},
		{Facts, "facts"},
		{PlayVars, "play_vars"},
		{RoleVars, "role_vars"},
		{BlockVars, "block_vars"},
		{TaskVars, "task_vars"},
		{Registered, "registered"},
		{RoleParams, "role_params"},
		{ExtraVars, "extra_vars"},
		{numLayers, "unknown"},
		{Layer(-1), "unknown"},
	}
	for _, c := range cases {
		if got := c.l.String(); got != c.want {
			t.Errorf("Layer(%d).String() = %q, want %q", c.l, got, c.want)
		}
	}
}

func TestSetReplacesWholeLayer(t *testing.T) {
	c := New()
	c.SetVar(PlayVars, "stale", "gone")
	c.Set(PlayVars, map[string]any{"a": 1, "b": 2})

	if _, ok := c.Get("stale"); ok {
		t.Fatal("Set should replace the whole layer, but the stale key survived")
	}
	if v, _ := c.Get("a"); v != 1 {
		t.Fatalf("a = %v, want 1", v)
	}
	if v, _ := c.Get("b"); v != 2 {
		t.Fatalf("b = %v, want 2", v)
	}
}

func TestSetIsACopy(t *testing.T) {
	c := New()
	src := map[string]any{"a": 1}
	c.Set(PlayVars, src)
	src["a"] = 2
	if v, _ := c.Get("a"); v != 1 {
		t.Fatalf("mutating the map passed to Set affected the Context: got %v, want 1", v)
	}
}

func TestSetVarOnZeroValueContext(t *testing.T) {
	// The zero value is documented as unusable, but SetVar defends
	// against a nil layer map (e.g. a Context built without New())
	// rather than panicking.
	var c Context
	c.SetVar(RoleDefaults, "x", "d")
	if v, _ := c.Get("x"); v != "d" {
		t.Fatalf("SetVar on a zero-value Context: Get(x) = %v, want d", v)
	}
}

func TestWhichNotSet(t *testing.T) {
	c := New()
	c.SetVar(RoleDefaults, "x", "d")
	if l, ok := c.Which("nowhere"); ok {
		t.Fatalf("Which(nowhere) = %v,%v want _,false", l, ok)
	}
}

func TestLayers(t *testing.T) {
	got := Layers()
	if len(got) != int(numLayers) {
		t.Fatalf("Layers() has %d entries, want %d", len(got), numLayers)
	}
	want := []Layer{
		RoleDefaults, Inventory, Facts, PlayVars, RoleVars,
		BlockVars, TaskVars, Registered, RoleParams, ExtraVars,
	}
	for i, l := range want {
		if got[i] != l {
			t.Errorf("Layers()[%d] = %v, want %v", i, got[i], l)
		}
	}
}
