package cmdroute

import (
	"libdb.so/arikawa/v4/api"
	"libdb.so/arikawa/v4/discord"
	"github.com/pkg/errors"
)

// BulkCommandsOverwriter is an interface that allows to overwrite all commands
// at once. Everything *api.Client will implement this interface, including
// *state.State.
type BulkCommandsOverwriter interface {
	CurrentApplication() (*discord.Application, error)
	BulkOverwriteCommands(appID discord.AppID, cmds []api.CreateCommandData) ([]discord.Command, error)
}

var _ BulkCommandsOverwriter = (*api.Client)(nil)

// OverwriteCommands overwrites all commands for the current application.
func OverwriteCommands(client BulkCommandsOverwriter, cmds []api.CreateCommandData) error {
	app, err := client.CurrentApplication()
	if err != nil {
		return errors.Wrap(err, "cannot get current app ID")
	}

	_, err = client.BulkOverwriteCommands(app.ID, cmds)
	return errors.Wrap(err, "cannot overwrite commands")
}
