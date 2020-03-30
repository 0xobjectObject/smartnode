package service

import (
    "github.com/urfave/cli"

    cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
)


// Register service commands
func RegisterCommands(app *cli.App, name string, aliases []string) {
    app.Commands = append(app.Commands, cli.Command{
        Name:      name,
        Aliases:   aliases,
        Usage:     "Manage Rocket Pool service",
        Subcommands: []cli.Command{

            // Start the Rocket Pool service
            cli.Command{
                Name:      "start",
                Aliases:   []string{"s"},
                Usage:     "Start the Rocket Pool service",
                UsageText: "rocketpool service start",
                Action: func(c *cli.Context) error {

                    // Validate arguments
                    if err := cliutils.ValidateArgs(c, 0, nil); err != nil {
                        return err
                    }

                    // Run command
                    return nil

                },
            },

            // Pause the Rocket Pool service
            cli.Command{
                Name:      "pause",
                Aliases:   []string{"p"},
                Usage:     "Pause the Rocket Pool service",
                UsageText: "rocketpool service pause",
                Action: func(c *cli.Context) error {

                    // Validate arguments
                    if err := cliutils.ValidateArgs(c, 0, nil); err != nil {
                        return err
                    }

                    // Run command
                    return nil

                },
            },

            // Stop the Rocket Pool service
            cli.Command{
                Name:      "stop",
                Aliases:   []string{"o"},
                Usage:     "Stop the Rocket Pool service",
                UsageText: "rocketpool service stop",
                Action: func(c *cli.Context) error {

                    // Validate arguments
                    if err := cliutils.ValidateArgs(c, 0, nil); err != nil {
                        return err
                    }

                    // Run command
                    return nil

                },
            },

            // Scale the Rocket Pool service
            cli.Command{
                Name:      "scale",
                Aliases:   []string{"c"},
                Usage:     "Scale the Rocket Pool service",
                UsageText: "rocketpool service scale",
                Action: func(c *cli.Context) error {

                    // Run command
                    return nil

                },
            },

            // View the Rocket Pool service logs
            cli.Command{
                Name:      "logs",
                Aliases:   []string{"l"},
                Usage:     "View the Rocket Pool service logs",
                UsageText: "rocketpool service logs",
                Action: func(c *cli.Context) error {

                    // Run command
                    return nil

                },
            },

            // View the Rocket Pool service resource stats
            cli.Command{
                Name:      "stats",
                Aliases:   []string{"t"},
                Usage:     "View the Rocket Pool service resource stats",
                UsageText: "rocketpool service stats",
                Action: func(c *cli.Context) error {

                    // Validate arguments
                    if err := cliutils.ValidateArgs(c, 0, nil); err != nil {
                        return err
                    }

                    // Run command
                    return nil

                },
            },

        },
    })
}

