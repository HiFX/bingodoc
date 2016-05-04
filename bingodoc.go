package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	bingoParser "github.com/hifx/bingodoc/parser"

	"fmt"
	"strings"
)

var output = flag.String("output", "swagger.json", "Output (path) for the generated file(s)")
var handlerPackage = flag.String("handlerPackage", "", "The package that implements the Handlers, relative to $GOPATH/src")

func main() {

	flag.Parse()

	if *handlerPackage == "" {
		fmt.Println("handlerPackage is missing")
		os.Exit(1)
	}
	if *output == "" {
		fmt.Println("output is missing")
		os.Exit(1)
	}

	//Create full file path
	wd, _ := os.Getwd()
	*handlerPackage = filepath.Clean(wd + "/" + *handlerPackage)

	parser := bingoParser.NewParser(*handlerPackage)

	//Take each api(.go) controller file from the apiPackage folder
	files, err := ioutil.ReadDir(*handlerPackage)
	if err != nil {
		log.Fatal(err)
		fmt.Println("Error in reading the directory")
		os.Exit(1)
	}

	for _, file := range files {
		fleName := file.Name()
		if strings.Contains(fleName, ".") {
			if strings.EqualFold(fleName[strings.LastIndex(fleName, "."):], ".go") {
				parser.ParseRequestParametersIntoStruct(*handlerPackage + "/" + file.Name())
			}
		}
	}

	//generating struct.go file in handler package folder
	parser.GenerateStructFile()
	defer parser.DeleteStructFile()

	//executing go-swagger
	buildCmd := exec.Command("go", "build")
	er := buildCmd.Run()
	if er != nil {
		fmt.Println("er ", er.Error())
		os.Exit(1)
	}

	//executing go-swagger
	swaggerCmd := exec.Command("swagger", "generate", "spec", "-o", *output)
	er = swaggerCmd.Run()
	if er != nil {
		fmt.Println("er ", er.Error())
		os.Exit(1)
	}

}
