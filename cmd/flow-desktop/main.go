// Command flow-desktop is the workspace overlay: a Wails v3 window over the
// in-process desktop.Workspace handler. It holds no truth of its own; every read
// aggregates the registered `.flow/` trees and every write resolves to one
// project (adr-0001, adr-0002). Deleting this binary loses pixels, nothing else.
package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"os"

	"github.com/juststeveking/flow/internal/core"
	"github.com/juststeveking/flow/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	reg := loadRegistry()
	ws := desktop.NewWorkspace(reg, nil)
	if err := ws.Reload(); err != nil {
		log.Fatal(err)
	}
	if err := ws.StartWatching(context.Background()); err != nil {
		log.Fatal(err)
	}

	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "flow",
		Description: "flow workspace overlay",
		Assets:      application.AssetOptions{Handler: ws.Handler(sub)},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "flow",
		Width:  1100,
		Height: 760,
		URL:    "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// loadRegistry reads the registry, and on first run seeds it with the current
// project if the working directory is inside a `.flow/` tree, so launching from a
// project just works.
func loadRegistry() *desktop.Registry {
	path, err := desktop.DefaultRegistryPath()
	if err != nil {
		log.Fatal(err)
	}
	if env := os.Getenv("FLOW_REGISTRY"); env != "" {
		path = env
	}
	reg, err := desktop.LoadRegistry(path)
	if err != nil {
		log.Fatal(err)
	}
	if len(reg.Roots) == 0 {
		if cwd, err := os.Getwd(); err == nil {
			if root, ok := core.FindRoot(cwd); ok {
				if p, err := core.LoadProject(root); err == nil && p.ProjectID() != "" {
					if reg.Add(root, p.ProjectID()) == nil {
						_ = reg.Save()
					}
				}
			}
		}
	}
	return reg
}
