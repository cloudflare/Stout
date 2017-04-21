package google

import (
	"github.com/eagerio/Stout/src/providers/google/fs"
	"github.com/eagerio/Stout/src/types"
)

func (cl *client) CreateFS(g types.GlobalFlags, c types.CreateFlags) (string, error) {
	err := fs.CreateFS(gclient, ctx, g.Domain, cl.ProjectID, cl.Location)
	if err != nil {
		return "", err
	}

	return "c.storage.googleapis.com", nil
}

func (a *client) FSProviderFuncs(g types.GlobalFlags) (types.FSProviderFunctions, error) {
	return fs.FSProviderFuncs(gclient, ctx, g.Domain)
}
