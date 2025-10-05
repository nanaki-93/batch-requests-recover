package cmd

import (
	"batchRequestsRecover/internal/model"
	"batchRequestsRecover/internal/service"
	"batchRequestsRecover/internal/util"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func Run() {
	args := checkAndParseArgs()

	config := loadConfig(args.ConfigFilePath)

	parserService := service.NewParserService(*config)
	processService := service.NewProcessService(*config, *args)

	fmt.Printf("Processing inputFile: %s\n", args.CSVFilePath)

	// Parse CSV records
	records, err := parserService.ReadAndParse(args.CSVFilePath)
	if err != nil {
		fmt.Println("Error reading CSV:", err)
		return
	}

	respList, errList, err := processService.ProcessAll(records)
	if err != nil {
		fmt.Println("Error processing records:", err)
		return
	}

	util.WriteResponses(args.CSVFilePath, errList, ".err")
	util.WriteResponses(args.CSVFilePath, respList, ".resp")

}

func checkAndParseArgs() *model.CommandLineArgs {
	csvFilePath := flag.String("inputFile", "", "Path to CSV inputFile")
	configFilePath := flag.String("configPath", "config.json", "Path to config file, default is config.json")
	dryRun := flag.Bool("dry", true, "Dry run")
	sleep := flag.Int("sleep", 1, "Sleep seconds between requests")

	flag.Parse()

	if *csvFilePath == "" {
		println(" inputFile is required")
		os.Exit(1)
	}
	return &model.CommandLineArgs{
		CSVFilePath:    *csvFilePath,
		ConfigFilePath: *configFilePath,
		DryRun:         *dryRun,
		SleepSeconds:   *sleep,
	}
}

func loadConfig(configFileName string) *model.Config {
	config := model.Config{}
	file, err := os.ReadFile(configFileName)

	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return &model.Config{}
	}
	return &config
}
