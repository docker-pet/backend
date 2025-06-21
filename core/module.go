package core

import (
	"fmt"
)

type Module interface {
	Name() string
	Deps() []string
	Init(ctx *AppContext, cfg any) error
}

type ModuleEntry struct {
	Module Module
	Config any
}

var moduleRegistry = map[string]*ModuleEntry{}

func RegisterModule(m Module, config any) {
	moduleRegistry[m.Name()] = &ModuleEntry{Module: m, Config: config}
}

func InitModules(ctx *AppContext) error {
	visited := map[string]bool{}

	var initModule func(name string) error
	initModule = func(name string) error {
		if visited[name] {
			return nil
		}
		entry, ok := moduleRegistry[name]
		if !ok {
			return fmt.Errorf("module not found: %s", name)
		}
		for _, dep := range entry.Module.Deps() {
			if err := initModule(dep); err != nil {
				return err
			}
		}
		if err := entry.Module.Init(ctx, entry.Config); err != nil {
			return err
		}
		if ctx.Modules == nil {
			ctx.Modules = map[string]Module{}
		}
		ctx.Modules[name] = entry.Module
		visited[name] = true
		return nil
	}

	for name := range moduleRegistry {
		if err := initModule(name); err != nil {
			return err
		}
	}
	return nil
}