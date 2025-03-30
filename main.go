//go:build linux

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var containersDirectory string = "./containers"
var rootFileSystemTarball string = "./ubuntu-base-22.04-base-amd64.tar.gz"

func init() {
	err := os.MkdirAll(containersDirectory, 0700)
	if err != nil {
		log.Fatal("Failed to create containers directory during initialization: ", err)
	}

	fmt.Println("===== Container Runtime Initialized =====")
	fmt.Println("Container storage location:", containersDirectory)
	fmt.Println("Using root filesystem:", rootFileSystemTarball)
	fmt.Println("=========================================")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Error: You must provide a command! Try 'run' to create a container or 'ps' to list containers.")
	}

	userCommand := os.Args[1]

	switch userCommand {
	case "run":
		fmt.Println("Starting container creation process...")
		handleContainerCreation(false)
	case "_child":
		fmt.Println("Initializing container environment...")
		handleContainerCreation(true)
	case "ps":
		fmt.Println("Listing all containers...")
		listAllContainers()
	default:
		log.Fatal("Error: Unknown command '" + userCommand + "'. Valid commands are 'run' (to create a container) and 'ps' (to list containers).")
	}
}

func handleContainerCreation(isChildProcess bool) {
	var volumeMappings []string
	var commandArguments []string

	fmt.Println("Parsing command arguments:", os.Args[2:])

	for argumentIndex := 2; argumentIndex < len(os.Args); argumentIndex++ {
		currentArgument := os.Args[argumentIndex]
		if strings.HasPrefix(currentArgument, "-v=") {
			volumeMapping := strings.TrimPrefix(currentArgument, "-v=")
			volumeMappings = append(volumeMappings, volumeMapping)
			fmt.Println("Detected volume mapping:", volumeMapping)
		} else {
			commandArguments = append(commandArguments, currentArgument)
		}
	}

	fmt.Println("Volume mappings:", volumeMappings)
	fmt.Println("Command arguments:", commandArguments)

	startContainer(commandArguments, volumeMappings, isChildProcess)
}

func startContainer(commandArgs []string, volumeMappings []string, isChildProcess bool) {
	if len(commandArgs) == 0 {
		log.Fatal("Error: You must specify a command to run in the container! Example: ./container run /bin/bash")
	}

	var executableToRun string
	var executableArguments []string

	if isChildProcess {
		fmt.Println("Child process: preparing to execute user command inside container")
		executableToRun = commandArgs[0]
		if len(commandArgs) > 1 {
			executableArguments = commandArgs[1:]
		} else {
			executableArguments = []string{}
		}
		fmt.Println("Child: Will execute:", executableToRun, "with arguments:", executableArguments)
	} else {
		fmt.Println("Parent process: preparing to create container environment")
		ourExecutablePath, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to get our own executable path: ", err)
		}
		executableToRun = ourExecutablePath
		executableArguments = append([]string{"_child"}, volumeMappings...)
		executableArguments = append(executableArguments, commandArgs...)
		fmt.Println("Parent: Will execute:", executableToRun, "with arguments:", executableArguments)
	}

	commandToExecute := exec.Command(executableToRun, executableArguments...)
	commandToExecute.Stdin = os.Stdin
	commandToExecute.Stdout = os.Stdout
	commandToExecute.Stderr = os.Stderr

	if isChildProcess {
		containerId := "container-" + generateRandomIdentifier(24)
		fmt.Println("\n===== CONTAINER SETUP START =====")
		fmt.Println("Container ID:", containerId)
		syscall.Sethostname([]byte(containerId))
		containerRootFsPath := filepath.Join(containersDirectory, containerId)
		extractRootFilesystem(containerRootFsPath, rootFileSystemTarball)
		fmt.Println("Root filesystem extracted successfully")
		changeContainerRoot(containerRootFsPath)
		syscall.Mount("proc", "/proc", "proc", 0, "")
		fmt.Println("\n===== CONTAINER READY =====")
		fmt.Println("Container ID:", containerId)
		fmt.Println("Container PID:", os.Getpid())
	}

	executionError := commandToExecute.Run()
	exitCode := commandToExecute.ProcessState.ExitCode()
	fmt.Println("\nCommand execution completed with exit code:", exitCode)
	os.Exit(exitCode)
}

func listAllContainers() {
	fmt.Println("\n===== CONTAINER LISTING =====")
	containerEntries, err := os.ReadDir(containersDirectory)
	if err != nil {
		log.Fatal("Failed to read containers directory: ", err)
	}
	if len(containerEntries) == 0 {
		fmt.Println("No containers found.")
		return
	}
	fmt.Println("CONTAINER ID\t\t\tCREATION TIME")
	fmt.Println("--------------------------------------------")
	for _, containerEntry := range containerEntries {
		containerInfo, err := containerEntry.Info()
		if err != nil {
			fmt.Printf("Warning: Could not get info for container '%s': %v\n", containerEntry.Name(), err)
			continue
		}
		fmt.Printf("%s\t%s\n", containerEntry.Name(), containerInfo.ModTime().Format(time.UnixDate))
	}
}

func generateRandomIdentifier(length int) string {
	const characterSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	resultCharacters := make([]byte, length)
	for i := 0; i < length; i++ {
		resultCharacters[i] = characterSet[rand.Intn(len(characterSet))]
	}
	return string(resultCharacters)
}

func extractRootFilesystem(destinationPath string, tarballPath string) {
	err := os.MkdirAll(destinationPath, 0700)
	if err != nil {
		log.Fatal("Failed to create root filesystem directory: ", err)
	}
	tarCommand := exec.Command("tar", "-xzf", tarballPath, "-C", destinationPath)
	if err := tarCommand.Run(); err != nil {
		log.Fatal("Failed to extract root filesystem tarball: ", err)
	}
}

func changeContainerRoot(newRootPath string) {
	syscall.Mount(newRootPath, newRootPath, "", syscall.MS_BIND, "")
}
