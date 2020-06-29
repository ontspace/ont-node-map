package main

import (
	"map/p2pserver"
	"map/storage"
	"map/web"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	alog "github.com/ontio/ontology-eventbus/log"
	"github.com/ontio/ontology/cmd"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/log"
	"github.com/urfave/cli"
)

func main() {
	storage.InitNodeDb()
	defer storage.CloseNodeDb()
	if err := setupAPP().Run(os.Args); err != nil {
		cmd.PrintErrorMsg(err.Error())
		os.Exit(1)
	}
}

func setupAPP() *cli.App {
	app := cli.NewApp()
	app.Usage = "Ontology Node Map CLI"
	app.Action = Start
	app.Version = "1.0.0"
	app.Copyright = "Copyright in 2019 @FYZ"
	app.Commands = []cli.Command{}
	app.Flags = []cli.Flag{
		//common setting
		cli.StringFlag{
			Name:   "config",
			Usage:  "Genesis block config `<file>`. If doesn't specifies, use main net config as default.",
			Hidden: true,
		},
		utils.LogLevelFlag,
		cli.UintFlag{
			Name:  "port",
			Usage: "Web App Port",
			Value: 8888,
		},
		cli.BoolFlag{
			Name:  "disablecors",
			Usage: "disable cors",
		},
		utils.NetworkIdFlag,
		utils.NodePortFlag,
		cli.UintFlag{
			Name:  "max-conn-in-bound",
			Usage: "Max connection `<number>` in bound",
			Value: 10240,
		},
		cli.UintFlag{
			Name:  "max-conn-out-bound",
			Usage: "Max connection `<number>` out bound",
			Value: 10240,
		},
		utils.MaxConnInBoundForSingleIPFlag,
	}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	return app
}

func initLog(ctx *cli.Context) {
	//init log module
	logLevel := ctx.GlobalInt(utils.GetFlagName(utils.LogLevelFlag))
	alog.InitLog(log.PATH)
	log.InitLog(logLevel, log.PATH, log.Stdout)
}

func Start(ctx *cli.Context) {
	initLog(ctx)

	log.Infof("ontology version %s", config.Version)

	setMaxOpenFiles()

	_, err := initConfig(ctx)
	if err != nil {
		log.Errorf("initConfig error: %s", err)
		return
	}

	p2p, err := p2pserver.NewServer(nil)
	if err != nil {
		log.Errorf("instance p2p server err: %v", err)
		return
	}
	if err := p2p.Start(); err != nil {
		log.Errorf("start p2p server err: %v", err)
		return
	}
	p2p.WaitForPeersStart()
	log.Infof("P2P init success")

	port := ctx.Uint("port")
	disableCors := ctx.Bool("disablecors")
	err = web.StartRestServer(port, disableCors)
	if err != nil {
		log.Error("start rest server failed", err)
		return
	}

	waitToExit()
}

func setMaxOpenFiles() {
	max, err := fdlimit.Maximum()
	if err != nil {
		log.Errorf("failed to get maximum open files: %v", err)
		return
	}
	_, err = fdlimit.Raise(uint64(max))
	if err != nil {
		log.Errorf("failed to set maximum open files: %v", err)
		return
	}
}

func initConfig(ctx *cli.Context) (*config.OntologyConfig, error) {
	//init ontology config from cli
	cfg, err := cmd.SetOntologyConfig(ctx)
	if err != nil {
		return nil, err
	}
	log.Infof("Config init success")
	return cfg, nil
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			log.Infof("Ontology received exit signal: %v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
