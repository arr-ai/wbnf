package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type VersionTags struct {
	Version   string
	GitCommit string
	BuildDate string
	BuildOS   string
}

func Main(info VersionTags) {
	app := cli.NewApp()
	// logrus.SetLevel(logrus.InfoLevel)

	app.EnableBashCompletion = true

	app.Name = "ωBNF"
	app.Usage = "the ultimate grammar helper app"
	app.Version = info.Version

	app.Commands = []cli.Command{testCommand, genCommand}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}
