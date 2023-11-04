package cmdroute

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
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
		return fmt.Errorf("cannot get current app ID: %w", err)
	}

	if _, err = client.BulkOverwriteCommands(app.ID, cmds); err != nil {
		return fmt.Errorf("cannot overwrite commands: %w", err)
	}

	return nil
}
