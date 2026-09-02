# vars

Ansible variable precedence engine: facts, defaults, host/group vars, extra-vars.

Part of [go-ansible](https://github.com/go-ansible) — a pure-Go (CGO=0),
functional-parity port of [Ansible](https://www.ansible.com/).

[![CI](https://github.com/go-ansible/vars/actions/workflows/ci.yml/badge.svg)](https://github.com/go-ansible/vars/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-ansible/vars.svg)](https://pkg.go.dev/github.com/go-ansible/vars)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue.svg)](LICENSE)

## Usage

```go
ctx := vars.New()
ctx.Set(vars.Inventory, hostVars)
ctx.SetVar(vars.TaskVars, "retries", 3)
ctx.SetVar(vars.ExtraVars, "env", "prod") // -e / --extra-vars — always wins

merged := ctx.Merged()          // low-to-high across all ten layers
val, ok := ctx.Get("env")       // "prod"
layer, ok := ctx.Which("env")   // vars.ExtraVars
```

`Layers()` lists the ladder in precedence order (`RoleDefaults` lowest through
`ExtraVars` highest); `Child()` derives a scoped context (e.g. per-block/per-task)
that inherits its parent's merged values without mutating them.
