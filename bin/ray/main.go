package main

import (
	"fmt"
	"os"

	"github.com/marco79423/ray/pkg/command/publish"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "ray"
	app.Usage = "SBK 後端小幫手"

	app.Commands = []*cli.Command{
		publish.Command(),
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v", err)
	}
}
