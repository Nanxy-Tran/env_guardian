package main

import (
	"bufio"
	"fmt"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

var frameworks = make(map[string][]string)
var reactNativeIgnores = []string{".gitignore", ".git", ".idea", ".jest", ".codeclimate.yml", "node_modules", "android/", "ios/", "coverage/", "cypress", ".svg"}
var laravelIgnores = []string{".gitignore", ".git", "node_modules", "vendor"}

type ScanResult []string

var results = ScanResult{}

func main() {

	app := &cli.App{
		Name:  "missing_env",
		Usage: "make an explosive entrance",

		Commands: []*cli.Command{

			{
				Name:  "scan_env",
				Usage: "complete a task on the list",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "framework",
						Value: "laravel",
						Usage: "framework to catch evn",
					},
					&cli.StringFlag{
						Name:  "path",
						Value: ".env.example",
						Usage: "env file absolute path",
					},
				},
				Action: func(cliContext *cli.Context) error {
					framework := cliContext.String("framework")
					envPath := cliContext.String("path")
					fmt.Println("Scanning env with ", framework, ", ", envPath)
					run(framework, envPath)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func run(framework string, envPath string) {
	start := time.Now()
	frameworks["react-native"] = reactNativeIgnores
	frameworks["laravel"] = laravelIgnores

	envs, _ := parseEnvFile(envPath)

	filesChan := make(chan string)
	var invalidLineChans chan (<-chan string)
	invalidLineChans = make(chan (<-chan string), 100)

	go scanFolder("./", filesChan)

	go func() {
		defer close(invalidLineChans)
		for path := range filesChan {
			invalidLineChans <- checkLines(path, envs)
		}
	}()

	for line := range invalidLineChans {
		select {
		case authorsString := <-line:
			countInvalidTranslation(authorsString)
		}
	}

	for line := range invalidLineChans {
		select {
		case authorsString := <-line:
			countInvalidTranslation(authorsString)
		default:

		}
	}

	printResult(results)
	fmt.Printf("Executed in %s", time.Since(start))
}

func scanFolder(root string, fileChan chan<- string) {
	files, err := ioutil.ReadDir(root)

	if err != nil {
		log.Fatal(err.Error())
	}

FilesLoop:
	for _, file := range files {
		var filePath string

		if root == "./" {
			filePath = root + file.Name()
		} else {
			filePath = root + "/" + file.Name()
		}

		for _, ignore := range frameworks["laravel"] {
			if strings.Contains(filePath, ignore) {
				continue FilesLoop
			}
		}

		if file.IsDir() {
			scanFolder(filePath, fileChan)
		} else {
			fileChan <- filePath
		}
	}

	if root == "./" {
		close(fileChan)
	}
}
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err, "error happened")
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func checkLines(filePath string, envs []string) <-chan string {
	lineChan := make(chan string, 50)

	fileLines, err := readLines(filePath)

	if err != nil {
		defer close(lineChan)
		return lineChan
	}

	go func() {
		defer close(lineChan)
		regex := regexp.MustCompile(envRegex)
		for _, line := range fileLines {
			quotedStrings := regex.FindAllString(line, -1)
			for _, quotedString := range quotedStrings {
				variable := strings.ReplaceAll(strings.ReplaceAll(quotedString, "'", ""), "env(", "")
				if contains(envs, variable) {
					continue
				} else {
					fmt.Println(variable, " is missing in env")
				}
			}
		}
	}()

	return lineChan
}

func countInvalidTranslation(filePath string) {
	if filePath == "" {
		return
	}
	results = append(results, filePath)
}

func printResult(results ScanResult) {
	if len(results) == 0 {
		fmt.Println("No hardcode found !")
		return
	}
	for _, filePath := range results {
		fmt.Println("Raw string File: ", filePath)
	}
}

var envRegex = "(env\\()['][^\"']*[']"

func parseEnvFile(envPath string) ([]string, error) {
	file, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var variables []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		variable := strings.Split(line, "=")[0]
		if strings.Contains(variable, "#") || variable == "" {
			continue
		}
		variables = append(variables, variable)
	}
	return variables, scanner.Err()
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
